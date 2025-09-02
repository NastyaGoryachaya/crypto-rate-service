package bot

import (
	"context"
	"strings"
	"time"

	"log/slog"

	"gopkg.in/telebot.v4"
)

// scheduler - планировщик рассылок для Telegram-бота.
type scheduler struct {
	bot         *telebot.Bot
	rates       RatesReader
	subs        SubscriptionStore
	checkPeriod time.Duration
	logger      *slog.Logger
}

func newScheduler(bot *telebot.Bot, r RatesReader, s SubscriptionStore, period time.Duration, logger *slog.Logger) *scheduler {
	if period <= 0 {
		period = time.Minute
	}
	logger.Debug("bot scheduler configured", slog.Duration("period", period))
	return &scheduler{bot: bot, rates: r, subs: s, checkPeriod: period, logger: logger}
}

// Run - основной цикл: раз в checkPeriod проверяем, кому пора отправить сообщение.
func (s *scheduler) run(ctx context.Context) {
	s.logger.Info("bot scheduler started", slog.Duration("period", s.checkPeriod))
	t := time.NewTicker(s.checkPeriod)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("bot scheduler stopped")
			return
		case now := <-t.C:
			s.logger.Debug("scheduler tick started", slog.Time("now", now))
			started := time.Now()
			s.tick(ctx, now)
			s.logger.Debug("scheduler tick completed", slog.Duration("duration", time.Since(started)))
		}
	}
}

// Tick - одна итерация рассылки: находим пользователей, собираем сообщение и отправляем.
func (s *scheduler) tick(ctx context.Context, now time.Time) {
	s.logger.Debug("tick: loading due subscriptions", slog.Time("now", now))
	chatIDs, err := s.subs.Due(ctx, now)
	if err != nil {
		s.logger.Error("failed to fetch due subscriptions", slog.Any("err", err))
		return
	}
	s.logger.Debug("tick: due subscriptions loaded", slog.Int("count", len(chatIDs)))
	if len(chatIDs) == 0 {
		s.logger.Debug("tick: no due subscriptions")
		return
	}

	// Получаем текущие курсы. Ставлю небольшой timeout, чтобы не блокировать надолго.
	rCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	rStarted := time.Now()
	rates, err := s.rates.GetCurrencyRates(rCtx)
	if err != nil {
		s.logger.Error("tick: failed to fetch rates", slog.Any("err", err))
		return
	}
	if len(rates) == 0 {
		s.logger.Warn("tick: fetched 0 rates")
		return
	}
	s.logger.Debug("tick: rates fetched", slog.Int("count", len(rates)), slog.Duration("duration", time.Since(rStarted)))

	msg := buildRatesMessage(rates)

	for _, id := range chatIDs {
		s.logger.Debug("tick: sending message", slog.Int64("chat_id", id))
		if _, err := s.bot.Send(&telebot.Chat{ID: id}, msg); err != nil {
			s.logger.Error("tick: send failed", slog.Int64("chat_id", id), slog.Any("err", err))
			continue
		}
		s.logger.Debug("tick: send ok", slog.Int64("chat_id", id))
		if err := s.subs.MarkSent(ctx, id, now); err != nil {
			s.logger.Error("tick: mark sent failed", slog.Int64("chat_id", id), slog.Any("err", err))
		} else {
			s.logger.Debug("tick: marked subscription sent", slog.Int64("chat_id", id))
		}
	}
}

// buildRatesMessage - форматирует общий текст для рассылок по всем монетам.
func buildRatesMessage(list []RateDTO) string {
	var b strings.Builder
	for _, r := range list {
		b.WriteString(formatRateLine(r))
		b.WriteByte('\n')
	}
	return b.String()
}
