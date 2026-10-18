package sqlite

import (
	"context"
	"database/sql"
	"encoding/json/v2"
	"fmt"
	"time"
)

type StudentRepository struct {
	db      *sql.DB
	timeout time.Duration
}

func NewStudentRepository(db *sql.DB, timeout time.Duration) *StudentRepository {
	return &StudentRepository{
		db:      db,
		timeout: timeout,
	}
}

func (r *StudentRepository) ReplaceAll(ctx context.Context, students map[int64]string) (err error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	payload, err := json.Marshal(students)
	if err != nil {
		return fmt.Errorf("marshal students: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.ExecContext(ctx, `DELETE FROM students`); err != nil {
		return fmt.Errorf("delete students: %w", err)
	}

	const insert = `
		INSERT INTO students (user_id, name)
		SELECT cast(key AS integer), value
		FROM json_each(?)
	`

	if _, err := tx.ExecContext(ctx, insert, string(payload)); err != nil {
		return fmt.Errorf("insert students: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (r *StudentRepository) LoadAll(ctx context.Context) (map[int64]string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		SELECT user_id, name
		FROM students
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query students: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	students := make(map[int64]string)

	for rows.Next() {
		var (
			id   int64
			name string
		)

		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan student: %w", err)
		}

		students[id] = name
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate students: %w", err)
	}

	return students, nil
}
