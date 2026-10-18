package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type TargetRepository struct {
	db      *sql.DB
	timeout time.Duration
}

func NewTargetRepository(db *sql.DB, timeout time.Duration) *TargetRepository {
	return &TargetRepository{
		db:      db,
		timeout: timeout,
	}
}

func (r *TargetRepository) SetTarget(ctx context.Context, target domain.Target) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		INSERT INTO queue_target (id, chat_id, thread_id)
		VALUES (1, ?, ?)
		ON CONFLICT (id) DO UPDATE SET 
		    chat_id = excluded.chat_id, 
		    thread_id = excluded.thread_id
	`

	if _, err := r.db.ExecContext(ctx, query, target.ChatID, target.ThreadID); err != nil {
		return fmt.Errorf("upsert queue target: %w", err)
	}

	return nil
}

func (r *TargetRepository) Target(ctx context.Context) (domain.Target, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		SELECT chat_id, thread_id
		FROM queue_target
		WHERE id = 1
	`

	var target domain.Target

	err := r.db.QueryRowContext(ctx, query).Scan(&target.ChatID, &target.ThreadID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Target{}, domain.ErrTargetNotSet
	}

	if err != nil {
		return domain.Target{}, fmt.Errorf("query queue target: %w", err)
	}

	return target, nil
}
