package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionRepo struct {
	db *pgxpool.Pool
}

func NewSubscriptionRepo(db *pgxpool.Pool) *SubscriptionRepo {
	return &SubscriptionRepo{db: db}
}

// MarkEnabled включает/обновляет подписку для заданного chatID с указанным интервалом (в минутах).
// last_sent_at не трогаем — если была подписка ранее, оставляем как есть.
func (r *SubscriptionRepo) MarkEnabled(ctx context.Context, chatID int64, intervalMinutes int) error {
	query := `
	INSERT INTO subscriptions (chat_id, interval_minutes, enabled, last_sent_at)
	VALUES ($1, $2, TRUE, NULL)
	ON CONFLICT (chat_id)
	DO UPDATE SET interval_minutes = EXCLUDED.interval_minutes,
	              enabled = TRUE,
	              last_sent_at = NULL`
	_, err := r.db.Exec(ctx, query, chatID, intervalMinutes)
	return err
}

// MarkDisabled выключает подписку для chatID.
func (r *SubscriptionRepo) MarkDisabled(ctx context.Context, chatID int64) error {
	query := `UPDATE subscriptions SET enabled = FALSE WHERE chat_id = $1`
	_, err := r.db.Exec(ctx, query, chatID)
	return err
}

// FindDue возвращает chat_id, для которых наступило время отправки на момент now.
func (r *SubscriptionRepo) FindDue(ctx context.Context, now time.Time) ([]int64, error) {
	query := `
	SELECT chat_id
	FROM subscriptions
	WHERE enabled = TRUE
	  AND (
		last_sent_at IS NULL
		OR EXTRACT(EPOCH FROM ($1::timestamptz - last_sent_at)) / 60 >= interval_minutes
	)`
	rows, err := r.db.Query(ctx, query, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

// MarkSent отмечает факт отправки для chatID.
func (r *SubscriptionRepo) MarkSent(ctx context.Context, chatID int64, at time.Time) error {
	query := `UPDATE subscriptions SET last_sent_at = $2 WHERE chat_id = $1`
	_, err := r.db.Exec(ctx, query, chatID, at)
	return err
}
