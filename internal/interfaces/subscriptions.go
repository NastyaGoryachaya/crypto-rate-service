package interfaces

import (
	"context"
	"time"
)

// Subscriptions — интерфейс для управления подписками
type Subscriptions interface {
	FindDue(ctx context.Context, now time.Time) ([]int64, error)
	MarkSent(ctx context.Context, chatID int64, at time.Time) error
	MarkEnabled(ctx context.Context, chatID int64, intervalMinutes int) error
	MarkDisabled(ctx context.Context, chatID int64) error
}

// SubscriptionCommander — интерфейс для команд хендлеров бота (вкл/выкл подписку).
type SubscriptionCommander interface {
	Enable(ctx context.Context, chatID int64, intervalMinutes int) error
	Disable(ctx context.Context, chatID int64) error
}

// SubscriptionDispatcher — интерфейс для планировщика бота (рассылка сообщений).
type SubscriptionDispatcher interface {
	DispatchDue(ctx context.Context) (sent int, err error)
}
