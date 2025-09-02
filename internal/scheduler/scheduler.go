package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/NastyaGoryachaya/crypto-rate-service/internal/service/fetch"
)

type Scheduler struct {
	fetchService fetch.Service
	interval     time.Duration
	logger       *slog.Logger
}

// NewScheduler — конструктор планировщика фонового обновления курсов
func NewScheduler(fetchService fetch.Service, interval time.Duration, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		fetchService: fetchService,
		interval:     interval,
		logger:       logger,
	}
}

// Start — запускает периодическое выполнение задачи до остановки контекста
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("scheduler started")
	s.logger.Debug("scheduler interval configured", slog.Duration("interval", s.interval))

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// первый запуск сразу
	s.runOnce(ctx)

	for {
		select {
		case <-ticker.C:
			s.runOnce(ctx)
		case <-ctx.Done():
			s.logger.Info("scheduler stopped")
			return
		}
	}
}

// runOnce — одна итерация: получить курсы и сохранить их в БД
func (s *Scheduler) runOnce(ctx context.Context) {
	s.logger.Debug("tick: running fetch cycle")
	if err := s.fetchService.FetchAndSaveCurrency(ctx); err != nil {
		s.logger.Error("tick: fetch failed", slog.Any("err", err))
	} else {
		s.logger.Debug("tick: fetch cycle completed")
	}
	s.logger.Debug("tick: completed")
}
