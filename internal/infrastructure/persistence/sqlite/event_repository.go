package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type EventRepository struct {
	db      *sql.DB
	timeout time.Duration
}

func NewEventRepository(db *sql.DB, timeout time.Duration) *EventRepository {
	return &EventRepository{
		db:      db,
		timeout: timeout,
	}
}

func (r *EventRepository) Create(ctx context.Context, event domain.Event) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		INSERT INTO events (chat_id, thread_id, title, status)
		VALUES (?, ?, ?, ?)
		RETURNING id
	`

	var id int64

	err := r.db.QueryRowContext(ctx, query, event.ChatID, event.ThreadID, event.Title, string(event.Status)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert event: %w", err)
	}

	return id, nil
}

func (r *EventRepository) Get(ctx context.Context, eventID int64) (domain.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		SELECT id, chat_id, thread_id, message_id, title, status
		FROM events
		WHERE id = ?
	`

	event, err := scanEvent(r.db.QueryRowContext(ctx, query, eventID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Event{}, domain.ErrEventNotFound
	}

	if err != nil {
		return domain.Event{}, fmt.Errorf("query event: %w", err)
	}

	return event, nil
}

func (r *EventRepository) ListRecent(ctx context.Context, limit int) ([]domain.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		SELECT id, chat_id, thread_id, message_id, title, status
		FROM events
		ORDER BY id DESC
		LIMIT ?
	`

	events, err := r.queryEvents(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent events: %w", err)
	}

	return events, nil
}

func (r *EventRepository) ListOpenIDs(ctx context.Context) ([]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		SELECT id
		FROM events
		WHERE status = 'open' AND message_id IS NOT NULL
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query open events: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var ids []int64

	for rows.Next() {
		var id int64

		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan event id: %w", err)
		}

		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate open events: %w", err)
	}

	return ids, nil
}

func (r *EventRepository) ListOpenWithUser(ctx context.Context, userID int64) ([]domain.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		SELECT e.id, e.chat_id, e.thread_id, e.message_id, e.title, e.status
		FROM events AS e
		JOIN queue_entries AS q
		    ON q.event_id = e.id
		WHERE e.status = 'open' AND e.message_id IS NOT NULL AND q.user_id = ?
		ORDER BY e.id
	`

	events, err := r.queryEvents(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("query open events with user: %w", err)
	}

	return events, nil
}

func (r *EventRepository) SetMessageID(ctx context.Context, eventID int64, messageID int) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		UPDATE events
		SET message_id = ?
		WHERE id = ?
	`

	if _, err := r.db.ExecContext(ctx, query, messageID, eventID); err != nil {
		return fmt.Errorf("update event message id: %w", err)
	}

	return nil
}

func (r *EventRepository) SetStatus(ctx context.Context, eventID int64, status domain.Status) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		UPDATE events
		SET status = ?,
		    closed_at = CASE WHEN ? = 'closed' THEN unixepoch() END
		WHERE id = ?
	`

	if _, err := r.db.ExecContext(ctx, query, string(status), string(status), eventID); err != nil {
		return fmt.Errorf("update event status: %w", err)
	}

	return nil
}

func (r *EventRepository) Delete(ctx context.Context, eventID int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const (
		deleteEntriesQuery = `
			DELETE FROM queue_entries
			WHERE event_id = ?
		`

		deleteEventQuery = `
			DELETE FROM events
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

	if _, err := tx.ExecContext(ctx, deleteEntriesQuery, eventID); err != nil {
		return fmt.Errorf("delete entries: %w", err)
	}

	if _, err := tx.ExecContext(ctx, deleteEventQuery, eventID); err != nil {
		return fmt.Errorf("delete event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *EventRepository) DeleteClosedBefore(ctx context.Context, before time.Time) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const (
		deleteEntriesQuery = `
			DELETE FROM queue_entries
			WHERE event_id IN (SELECT id FROM events WHERE status = 'closed' AND closed_at < ?)
		`

		deleteEventsQuery = `
			DELETE FROM events
			WHERE status = 'closed' AND closed_at < ?
		`
	)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, deleteEntriesQuery, before.Unix()); err != nil {
		return 0, fmt.Errorf("delete entries: %w", err)
	}

	res, err := tx.ExecContext(ctx, deleteEventsQuery, before.Unix())
	if err != nil {
		return 0, fmt.Errorf("delete events: %w", err)
	}

	deleted, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count deleted events: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return int(deleted), nil
}

func (r *EventRepository) queryEvents(ctx context.Context, query string, args ...any) ([]domain.Event, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rows.Close()
	}()

	var events []domain.Event

	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return events, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEvent(row rowScanner) (domain.Event, error) {
	var (
		event     domain.Event
		messageID sql.NullInt64
		status    string
	)

	if err := row.Scan(&event.ID, &event.ChatID, &event.ThreadID, &messageID, &event.Title, &status); err != nil {
		return domain.Event{}, err
	}

	event.MessageID = int(messageID.Int64)
	event.Status = domain.Status(status)

	return event, nil
}
