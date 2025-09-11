package subscription

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/interfaces"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/pkg/botfmt"
	"github.com/NastyaGoryachaya/crypto-rate-service/internal/pkg/utils"
	"gopkg.in/telebot.v4"
)

type Service struct {
	bot            *telebot.Bot
	repo           interfaces.Subscriptions
	cryptoProvider interfaces.CryptoProvider
	log            *slog.Logger
	fetchTimeout   time.Duration
}

func New(bot *telebot.Bot, repo interfaces.Subscriptions, cryptoProvider interfaces.CryptoProvider, log *slog.Logger) *Service {
	return &Service{
		bot:            bot,
		repo:           repo,
		cryptoProvider: cryptoProvider,
		log:            log,
		fetchTimeout:   4 * time.Second,
	}
}

// Enable включает авторассылку для чата.
// Идемпотентна: повторный вызов с теми же параметрами безопасен.
func (s *Service) Enable(ctx context.Context, chatID int64, intervalMinutes int) error {
	if intervalMinutes <= 0 {
		return errors.New("interval must be > 0")
	}
	if err := s.repo.MarkEnabled(ctx, chatID, intervalMinutes); err != nil {
		s.log.Error("subscriptions.enable failed",
			slog.Int64("chat_id", chatID),
			slog.Int("interval_min", intervalMinutes),
			slog.String("err", err.Error()))
		return err
	}
	s.log.Info("subscriptions.enable ok",
		slog.Int64("chat_id", chatID),
		slog.Int("interval_min", intervalMinutes))
	return nil
}

// Disable отключает авторассылку для чата.
// Идемпотентна: если уже выключена — ошибки нет.
func (s *Service) Disable(ctx context.Context, chatID int64) error {
	if err := s.repo.MarkDisabled(ctx, chatID); err != nil {
		s.log.Error("subscriptions.disable failed",
			slog.Int64("chat_id", chatID),
			slog.String("err", err.Error()))
		return err
	}
	s.log.Info("subscriptions.disable ok", slog.Int64("chat_id", chatID))
	return nil
}

// DispatchDue выполняет одну итерацию авторассылки:
//  1. Находит чаты, у которых истёк интервал (due).
//  2. Получает свежие курсы криптовалют.
//  3. Формирует компактное сообщение (строки line).
//  4. Отправляет его каждому due-чату.
//  5. Отмечает отправку в репозитории.
//
// Возвращает количество успешно отправленных сообщений.
func (s *Service) DispatchDue(ctx context.Context) (sent int, err error) {
	now := utils.NowFunc()
	s.log.Debug("subscriptions.loading_due", slog.Time("now", now))

	chatIDs, err := s.repo.FindDue(ctx, now)
	if err != nil {
		s.log.Error("subscriptions.find_due failed", slog.String("err", err.Error()))
		return 0, err
	}
	if len(chatIDs) == 0 {
		s.log.Debug("subscriptions.no_due")
		return 0, nil
	}

	// Получаем курсы с коротким таймаутом
	rCtx, cancel := context.WithTimeout(ctx, s.fetchTimeout)
	defer cancel()

	start := time.Now()
	rates, err := s.cryptoProvider.FetchRates(rCtx)
	if err != nil {
		s.log.Error("subscriptions.fetch_rates failed", slog.String("err", err.Error()))
		return 0, err
	}
	if len(rates) == 0 {
		s.log.Warn("subscriptions.empty_rates")
		return 0, nil
	}
	s.log.Debug("subscriptions.rates_fetched",
		slog.Int("count", len(rates)),
		slog.Duration("duration", time.Since(start)))

	// Используем форматтер из format.go
	var b strings.Builder
	for i, r := range rates {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(botfmt.FormatRateLine(r))
	}
	msg := b.String()

	sent = 0
	for _, chatID := range chatIDs {
		if _, err := s.bot.Send(&telebot.Chat{ID: chatID}, msg); err != nil {
			s.log.Error("subscriptions.send failed",
				slog.Int64("chat_id", chatID),
				slog.String("err", err.Error()))
			continue
		}
		if err := s.repo.MarkSent(ctx, chatID, now); err != nil {
			s.log.Error("subscriptions.mark_sent failed",
				slog.Int64("chat_id", chatID),
				slog.String("err", err.Error()))
			continue
		}
		sent++
	}
	s.log.Info("subscriptions.dispatch_done",
		slog.Int("due", len(chatIDs)),
		slog.Int("sent", sent))
	return sent, nil
}
