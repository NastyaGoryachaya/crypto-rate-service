package interfaces

import "context"

// SubscriptionDispatcher - Нужен шедулеру: просто дернуть рассылку.
type SubscriptionDispatcher interface {
	DispatchDue(ctx context.Context) (sent int, err error)
}

// SubscriptionCommander - Нужен хендлерам бота: команды управления подпиской.
type SubscriptionCommander interface {
	Enable(ctx context.Context, chatID int64, intervalMinutes int) error
	Disable(ctx context.Context, chatID int64) error
}

type SubscriptionsService interface{}
