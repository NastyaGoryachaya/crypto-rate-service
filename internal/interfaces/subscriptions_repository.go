package interfaces

import (
	"context"
	"time"
)

// Subscriptions — интерфейс для управления подписками
type Subscriptions interface {
	FindDue(ctx context.Context, now time.Time) ([]int64, error)
	MarkSent(ctx context.Context, chatID int64, at time.Time) error
	Enable(ctx context.Context, chatID int64, intervalMinutes int) error
	Disable(ctx context.Context, chatID int64) error
}
