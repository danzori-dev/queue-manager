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
		INSERT INTO queue_entries (event_id, user_id, username, sort_key)
		SELECT ?, ?, ?, coalesce(max(sort_key), 0) + 1
		FROM queue_entries
		WHERE event_id = ?
		ON CONFLICT (event_id, user_id) DO NOTHING
		RETURNING id
	`

	var id int64

	err := r.db.
		QueryRowContext(ctx, query, eventID, userID, username, eventID).
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
			AND sort_key <= (SELECT sort_key FROM queue_entries WHERE event_id = ? AND user_id = ?)
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
		ORDER BY sort_key
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

func (r *EntryRepository) SwapWithNext(ctx context.Context, eventID, userID int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const (
		keyQuery = `
			SELECT sort_key
			FROM queue_entries
			WHERE event_id = ? AND user_id = ?
		`

		nextQuery = `
			SELECT id, sort_key
			FROM queue_entries
			WHERE event_id = ? AND sort_key > ?
			ORDER BY sort_key
			LIMIT 1
		`

		setKeyByUser = `
			UPDATE queue_entries
			SET sort_key = ?
			WHERE event_id = ? AND user_id = ?
		`

		setKeyByID = `
			UPDATE queue_entries
			SET sort_key = ?
			WHERE id = ?
		`
	)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var key int64

	err = tx.
		QueryRowContext(ctx, keyQuery, eventID, userID).
		Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotInQueue
	}

	if err != nil {
		return fmt.Errorf("query entry: %w", err)
	}

	var nextID, nextKey int64

	err = tx.
		QueryRowContext(ctx, nextQuery, eventID, key).
		Scan(&nextID, &nextKey)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrLastInQueue
	}

	if err != nil {
		return fmt.Errorf("query next entry: %w", err)
	}

	if _, err := tx.ExecContext(ctx, setKeyByUser, nextKey, eventID, userID); err != nil {
		return fmt.Errorf("move entry down: %w", err)
	}

	if _, err := tx.ExecContext(ctx, setKeyByID, key, nextID); err != nil {
		return fmt.Errorf("move next entry up: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *EntryRepository) Prepend(ctx context.Context, eventID int64, entries []domain.Entry) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const (
		minKeyQuery = `
			SELECT coalesce(min(sort_key), 1)
			FROM queue_entries
			WHERE event_id = ?
		`

		upsertQuery = `
			INSERT INTO queue_entries (event_id, user_id, username, sort_key)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (event_id, user_id) DO UPDATE SET sort_key = excluded.sort_key
		`
	)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var minKey int64

	err = tx.
		QueryRowContext(ctx, minKeyQuery, eventID).
		Scan(&minKey)
	if err != nil {
		return fmt.Errorf("query min sort key: %w", err)
	}

	first := minKey - int64(len(entries))

	for i, e := range entries {
		if _, err := tx.ExecContext(ctx, upsertQuery, eventID, e.UserID, e.Username, first+int64(i)); err != nil {
			return fmt.Errorf("upsert entry: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
