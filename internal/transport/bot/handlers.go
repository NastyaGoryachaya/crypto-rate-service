package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/consts"
	errs "github.com/NastyaGoryachaya/crypto-rate-service/internal/errors"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/pkg/botfmt"
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
		list, err := b.svc.GetLatest(ctx)
		if err != nil {
			if errors.Is(err, errs.ErrPriceNotFound) {
				return c.Send("Данные о цене не найдены")
			}
			return c.Send("Внутренняя ошибка сервиса, попробуйте позже")
		}
		if len(list) == 0 {
			return c.Send("Данные о цене не найдены")
		}
		var bld strings.Builder
		for _, r := range list {
			bld.WriteString(botfmt.FormatRateLine(r))
			bld.WriteByte('\n')
		}
		return c.Send(bld.String())
	}

	symbol := args[0]
	symbol = strings.ToUpper(symbol)
	if !consts.IsTracked(symbol) {
		return c.Send("Монета не поддерживается. Доступны: BTC, ETH")
	}
	now := time.Now().UTC()
	from := now.Add(-24 * time.Hour)
	to := now

	latest, minV, maxV, pct, err := b.svc.GetLatestBySymbol(ctx, symbol, from, to)
	if err != nil {
		if errors.Is(err, errs.ErrCoinNotFound) {
			return c.Send("Валюта не найдена")
		}
		if errors.Is(err, errs.ErrPriceNotFound) {
			return c.Send("Данные о цене не найдены")
		}
		return c.Send("Внутренняя ошибка сервиса, попробуйте позже")
	}
	return c.Send(botfmt.FormatRateDetails(latest, minV, maxV, pct))
}

// handleStartAuto — включает авторассылку курсов для чата с указанным интервалом в минутах
func (b *Bot) handleStartAuto(c telebot.Context) error {
	b.logger.Debug("subscription: /startauto received",
		slog.Int64("chat_id", c.Chat().ID),
		slog.String("text", c.Text()),
		slog.Int("args_len", len(c.Args())),
	)

	args := c.Args()
	chatID := c.Chat().ID
	if len(args) != 1 {
		b.logger.Warn("subscription: /startauto wrong args",
			slog.Int64("chat_id", chatID),
			slog.Int("args_len", len(args)),
			slog.String("text", c.Text()),
		)
		return c.Send("Укажи интервал в минутах: /startauto 10")
	}
	mins, err := parseMinutes(args[0])
	if err != nil {
		b.logger.Warn("subscription: /startauto invalid interval",
			slog.Int64("chat_id", chatID),
			slog.String("arg", args[0]),
		)
		return c.Send("Некорректный интервал. Пример: /startauto 10")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := b.subs.Enable(ctx, chatID, mins); err != nil {
		return c.Send("Внутренняя ошибка сервиса, попробуйте позже")
	}
	b.logger.Debug("subscription: startauto enabled",
		slog.Int64("chat_id", chatID),
		slog.Int("interval_min", mins),
	)
	if err := c.Send(fmt.Sprintf("Автообновления включены! (каждые %d мин.) ", mins)); err != nil {
		b.logger.Error("subscription: /startauto confirm send failed",
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
		return c.Send("Внутренняя ошибка сервиса, попробуйте позже")
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
