package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type AdminRepository struct {
	db      *sql.DB
	timeout time.Duration
}

func NewAdminRepository(db *sql.DB, timeout time.Duration) *AdminRepository {
	return &AdminRepository{
		db:      db,
		timeout: timeout,
	}
}

func (r *AdminRepository) Add(ctx context.Context, userID int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		INSERT INTO admins (user_id)
		VALUES (?)
		ON CONFLICT (user_id) DO NOTHING
	`

	if _, err := r.db.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("insert admin: %w", err)
	}

	return nil
}

func (r *AdminRepository) Remove(ctx context.Context, userID int64) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		DELETE FROM admins
		WHERE user_id = ?
	`

	if _, err := r.db.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("delete admin: %w", err)
	}

	return nil
}

func (r *AdminRepository) List(ctx context.Context) ([]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	const query = `
		SELECT user_id
		FROM admins
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query admins: %w", err)
	}

	defer func() {
		_ = rows.Close()
	}()

	var ids []int64

	for rows.Next() {
		var id int64

		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan admin: %w", err)
		}

		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admins: %w", err)
	}

	return ids, nil
}
