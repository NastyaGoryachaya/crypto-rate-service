package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/ports/errcode"
	"gopkg.in/telebot.v4"
)

var ErrInvalidInterval = errors.New("invalid interval")

// handleStart — отправляет справку по доступным командам бота
func (b *Bot) handleStart(c telebot.Context) error {
	return c.Send("Привет! Доступные команды:\n" +
		"/rates - цены по всем валютам\n" +
		"/rates {symbol} - цена по конкретной валюте (BTC/ETH)\n" +
		"/startauto {минуты} - включить автообновления\n" +
		"/stopauto - отключить автообновления")
}

// handleRates — выводит курсы: без аргументов — все монеты, с аргументом символа — подробности по одной
func (b *Bot) handleRates(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := c.Args()
	if len(args) == 0 {
		list, err := b.rates.GetCurrencyRates(ctx)
		if err != nil {
			return c.Send(translateBotError(errcode.Internal))
		}
		if len(list) == 0 {
			return c.Send(translateBotError(errcode.NotFoundPrices))
		}
		var bld strings.Builder
		for _, r := range list {
			bld.WriteString(formatRateLine(r))
			bld.WriteByte('\n')
		}
		return c.Send(bld.String())
	}

	symbol := args[0]
	item, err := b.rates.GetCurrencyRateBySymbol(ctx, symbol)
	if err != nil {
		return c.Send(translateBotError(errcode.NotFoundCoins))
	}
	return c.Send(formatRateDetails(item))
}

// handleStartAuto — включает авторассылку курсов для чата с указанным интервалом в минутах
func (b *Bot) handleStartAuto(c telebot.Context) error {
	b.logger.Debug("bot: /startauto received",
		slog.Int64("chat_id", c.Chat().ID),
		slog.String("text", c.Text()),
		slog.Int("args_len", len(c.Args())),
	)

	args := c.Args()
	chatID := c.Chat().ID
	if len(args) != 1 {
		b.logger.Warn("bot: /startauto wrong args",
			slog.Int64("chat_id", chatID),
			slog.Int("args_len", len(args)),
			slog.String("text", c.Text()),
		)
		return c.Send("Укажи интервал в минутах: /startauto 10")
	}
	mins, err := parseMinutes(args[0])
	if err != nil {
		b.logger.Warn("bot: /startauto invalid interval",
			slog.Int64("chat_id", chatID),
			slog.String("arg", args[0]),
		)
		return c.Send("Некорректный интервал. Пример: /startauto 10")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := b.subs.Enable(ctx, chatID, mins); err != nil {
		return c.Send(translateBotError(errcode.Internal))
	}
	b.logger.Debug("bot: startauto enabled",
		slog.Int64("chat_id", chatID),
		slog.Int("interval_min", mins),
	)
	if err := c.Send(fmt.Sprintf("Автообновления включены! (каждые %d мин.) ", mins)); err != nil {
		b.logger.Error("bot: /startauto confirm send failed",
			slog.Int64("chat_id", chatID),
			slog.String("error", err.Error()),
		)
		return err
	}
	return nil
}

// handleStopAuto — отключает авторассылку курсов для текущего чата
func (b *Bot) handleStopAuto(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := b.subs.Disable(ctx, c.Chat().ID); err != nil {
		return c.Send(translateBotError(errcode.Internal))
	}
	return c.Send("Автообновления отключены!")
}

// parseMinutes — парсит строку с минутами и валидирует значение (> 0)
func parseMinutes(s string) (int, error) {
	s = strings.TrimSpace(s)
	m, err := strconv.Atoi(s)
	if err != nil || m <= 0 {
		return 0, ErrInvalidInterval
	}
	return m, nil
}
