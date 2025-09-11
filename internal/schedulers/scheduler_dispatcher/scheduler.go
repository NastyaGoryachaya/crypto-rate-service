package scheduler_dispatcher

import (
	"context"
	"log/slog"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/interfaces"
)

type Scheduler struct {
	svc         interfaces.SubscriptionDispatcher
	checkPeriod time.Duration
	logger      *slog.Logger
}

func NewScheduler(svc interfaces.SubscriptionDispatcher, period time.Duration, logger *slog.Logger) *Scheduler {
	if period <= 0 {
		period = time.Minute
	}
	logger.Debug("subscription schedulers configured", slog.Duration("period", period))
	return &Scheduler{svc: svc, checkPeriod: period, logger: logger}
}

// Run - основной цикл: раз в checkPeriod проверяем, кому пора отправить сообщение.
func (s *Scheduler) Run(ctx context.Context) {
	s.logger.Info("subscription schedulers started", slog.Duration("period", s.checkPeriod))
	t := time.NewTicker(s.checkPeriod)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			s.logger.Info("subscription schedulers stopped")
			return
		case <-t.C:
			s.logger.Debug("schedulers tick started")
			started := time.Now()
			s.tick(ctx)
			s.logger.Debug("schedulers tick completed", slog.Duration("duration", time.Since(started)))
		}
	}
}

// tick — одна итерация рассылки: планировщик просто дергает сервис.
func (s *Scheduler) tick(ctx context.Context) {
	s.logger.Debug("tick: started")
	started := time.Now()
	sent, err := s.svc.DispatchDue(ctx)
	if err != nil {
		s.logger.Error("tick: dispatch failed", slog.String("err", err.Error()))
	} else {
		s.logger.Info("tick: dispatch completed", slog.Int("sent", sent), slog.Duration("duration", time.Since(started)))
	}
}
