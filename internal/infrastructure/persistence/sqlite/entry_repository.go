package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type EntryRepository struct {
	db      *sql.DB
	timeout time.Duration
}

func NewEntryRepository(db *sql.DB, timeout time.Duration) *EntryRepository {
	return &EntryRepository{
		db:      db,
		timeout: timeout,
	}
}

func (r *EntryRepository) Add(ctx context.Context, eventID, userID int64, username string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		INSERT INTO queue_entries (event_id, user_id, username)
		VALUES (?, ?, ?)
		ON CONFLICT (event_id, user_id) DO NOTHING
		RETURNING id
	`

	var id int64

	err := r.db.
		QueryRowContext(ctx, query, eventID, userID, username).
		Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("insert entry: %w", err)
	}

	return true, nil
}

func (r *EntryRepository) Remove(ctx context.Context, eventID, userID int64) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		DELETE FROM queue_entries
		WHERE event_id = ? AND user_id = ?
		RETURNING id
	`

	var id int64

	err := r.db.
		QueryRowContext(ctx, query, eventID, userID).
		Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("delete entry: %w", err)
	}

	return true, nil
}

func (r *EntryRepository) Position(ctx context.Context, eventID, userID int64) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		SELECT count(*)
		FROM queue_entries
		WHERE event_id = ?
			AND id <= (SELECT id FROM queue_entries WHERE event_id = ? AND user_id = ?)
	`

	var position int

	err := r.db.
		QueryRowContext(ctx, query, eventID, eventID, userID).
		Scan(&position)
	if err != nil {
		return 0, fmt.Errorf("query position: %w", err)
	}

	return position, nil
}

func (r *EntryRepository) List(ctx context.Context, eventID int64) ([]domain.Entry, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		SELECT id, user_id, username
		FROM queue_entries
		WHERE event_id = ?
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var entries []domain.Entry

	for rows.Next() {
		entry := domain.Entry{EventID: eventID}

		if err := rows.Scan(&entry.ID, &entry.UserID, &entry.Username); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entries: %w", err)
	}

	return entries, nil
}
