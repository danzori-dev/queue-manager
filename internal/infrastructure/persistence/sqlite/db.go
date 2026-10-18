package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type Config struct {
	DSN          string
	AuthToken    string
	QueryTimeout time.Duration
}

func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	parsed, err := url.Parse(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	q := parsed.Query()
	q.Set("authToken", cfg.AuthToken)
	parsed.RawQuery = q.Encode()

	db, err := sql.Open("libsql", parsed.String())
	if err != nil {
		return nil, fmt.Errorf("sql open: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.QueryTimeout)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()

		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}
