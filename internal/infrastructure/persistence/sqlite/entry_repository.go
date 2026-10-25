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
		SELECT count(DISTINCT sort_key)
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
		SELECT user_id, username, sort_key
		FROM queue_entries
		WHERE event_id = ?
		ORDER BY sort_key, id
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
		var entry domain.Entry

		if err := rows.Scan(&entry.UserID, &entry.Username, &entry.SortKey); err != nil {
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
			SELECT min(sort_key)
			FROM queue_entries
			WHERE event_id = ? AND sort_key > ?
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

	var next sql.NullInt64

	err = tx.
		QueryRowContext(ctx, nextQuery, eventID, key).
		Scan(&next)
	if err != nil {
		return fmt.Errorf("query next place: %w", err)
	}

	if !next.Valid {
		return domain.ErrLastInQueue
	}

	if err := swapPlaces(ctx, tx, eventID, key, next.Int64); err != nil {
		return err
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

	places := domain.Places(entries)
	first := minKey - int64(len(places))

	for i, place := range places {
		for _, e := range place {
			if _, err := tx.ExecContext(ctx, upsertQuery, eventID, e.UserID, e.Username, first+int64(i)); err != nil {
				return fmt.Errorf("upsert entry: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *EntryRepository) SwapSortKeys(ctx context.Context, eventID, userA, userB int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		SELECT sort_key
		FROM queue_entries
		WHERE event_id = ? AND user_id = ?
	`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var keyA, keyB int64

	err = tx.
		QueryRowContext(ctx, query, eventID, userA).
		Scan(&keyA)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotInQueue
	}

	if err != nil {
		return fmt.Errorf("query first entry: %w", err)
	}

	err = tx.
		QueryRowContext(ctx, query, eventID, userB).
		Scan(&keyB)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotInQueue
	}

	if err != nil {
		return fmt.Errorf("query second entry: %w", err)
	}

	if err := swapPlaces(ctx, tx, eventID, keyA, keyB); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *EntryRepository) JoinPlace(ctx context.Context, eventID, userID int64, username string, partnerID int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const (
		keyQuery = `
			SELECT sort_key
			FROM queue_entries
			WHERE event_id = ? AND user_id = ?
		`

		sizeQuery = `
			SELECT count(*)
			FROM queue_entries
			WHERE event_id = ? AND sort_key = ?
		`

		upsertQuery = `
			INSERT INTO queue_entries (event_id, user_id, username, sort_key)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (event_id, user_id) DO UPDATE 
			    SET sort_key = excluded.sort_key,
			        username = excluded.username
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
		QueryRowContext(ctx, keyQuery, eventID, partnerID).
		Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrNotInQueue
	}

	if err != nil {
		return fmt.Errorf("query partner entry: %w", err)
	}

	var current int64

	err = tx.
		QueryRowContext(ctx, keyQuery, eventID, userID).
		Scan(&current)

	switch {
	case err == nil && current == key:
		return domain.ErrAlreadyOnPlace

	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("query entry: %w", err)
	}

	var size int

	if err := tx.QueryRowContext(ctx, sizeQuery, eventID, key).Scan(&size); err != nil {
		return fmt.Errorf("query place size: %w", err)
	}

	if size >= domain.MaxPlaceSize {
		return domain.ErrPlaceFull
	}

	if _, err := tx.ExecContext(ctx, upsertQuery, eventID, userID, username, key); err != nil {
		return fmt.Errorf("upsert entry: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func swapPlaces(ctx context.Context, tx *sql.Tx, eventID, keyA, keyB int64) error {
	const query = `
		UPDATE queue_entries
		SET sort_key = CASE sort_key
		    WHEN ? THEN ?
		    ELSE ?
		END
		WHERE event_id = ? AND sort_key IN (?, ?)
	`

	if _, err := tx.ExecContext(ctx, query, keyA, keyB, keyA, eventID, keyA, keyB); err != nil {
		return fmt.Errorf("swap places: %w", err)
	}

	return nil
}
