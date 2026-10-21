package cleaner

import (
	"context"
	"log/slog"
	"time"
)

const firstRunDelay = time.Minute

type Config struct {
	Retention time.Duration
	Interval  time.Duration
}

type Cleaner interface {
	CleanupClosedEvents(ctx context.Context, olderThan time.Duration) (int, error)
}

type Worker struct {
	cfg Config
	log *slog.Logger
}

func New(cfg Config, log *slog.Logger) *Worker {
	return &Worker{
		cfg: cfg,
		log: log,
	}
}

func (w *Worker) Run(ctx context.Context, c Cleaner) error {
	timer := time.NewTimer(firstRunDelay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-timer.C:
			w.cleanup(ctx, c)
			timer.Reset(w.cfg.Interval)
		}
	}
}

func (w *Worker) cleanup(ctx context.Context, c Cleaner) {
	deleted, err := c.CleanupClosedEvents(ctx, w.cfg.Retention)
	if err != nil {
		if ctx.Err() != nil {
			return
		}

		w.log.ErrorContext(
			ctx, "cleanup closed events",
			slog.Any("err", err),
		)

		return
	}

	w.log.InfoContext(
		ctx, "cleanup done",
		slog.Int("deleted_events", deleted),
	)
}
