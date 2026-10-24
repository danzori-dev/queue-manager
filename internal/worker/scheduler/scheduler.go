package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Renderer interface {
	RenderEvent(ctx context.Context, eventID int64) (time.Duration, error)
	OpenEventIDs(ctx context.Context) ([]int64, error)
}

type Config struct {
	Debounce    time.Duration
	MinInterval time.Duration
	RetryDelay  time.Duration
	MaxAttempts int
}

type Scheduler struct {
	cfg Config
	log *slog.Logger

	mu       sync.Mutex
	ctx      context.Context //nolint:containedctx // scheduler lifetime ctx
	renderer Renderer
	timers   map[int64]*time.Timer
	lastRun  map[int64]time.Time
	failures map[int64]int

	wg sync.WaitGroup
}

func New(cfg Config, log *slog.Logger) *Scheduler {
	return &Scheduler{
		cfg:      cfg,
		log:      log,
		timers:   make(map[int64]*time.Timer),
		lastRun:  make(map[int64]time.Time),
		failures: make(map[int64]int),
	}
}

func (s *Scheduler) Schedule(eventID int64) {
	s.schedule(eventID, s.cfg.Debounce)
}

func (s *Scheduler) Run(ctx context.Context, r Renderer) error {
	s.mu.Lock()
	s.ctx = ctx
	s.renderer = r
	s.mu.Unlock()

	ids, err := r.OpenEventIDs(ctx)
	if err != nil {
		return fmt.Errorf("list open events: %w", err)
	}

	for _, id := range ids {
		s.Schedule(id)
	}

	<-ctx.Done()

	s.mu.Lock()
	for id, t := range s.timers {
		if t.Stop() {
			s.wg.Done()
		}

		delete(s.timers, id)
	}

	s.mu.Unlock()

	s.wg.Wait()

	return nil
}

func (s *Scheduler) schedule(eventID int64, delay time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ctx == nil || s.ctx.Err() != nil {
		return
	}

	if _, ok := s.timers[eventID]; ok {
		return
	}

	if wait := s.cfg.MinInterval - time.Since(s.lastRun[eventID]); wait > delay {
		delay = wait
	}

	s.wg.Add(1)
	s.timers[eventID] = time.AfterFunc(
		delay, func() {
			defer s.wg.Done()

			s.fire(eventID)
		},
	)
}

func (s *Scheduler) fire(eventID int64) {
	s.mu.Lock()
	delete(s.timers, eventID)
	s.lastRun[eventID] = time.Now()
	ctx, r := s.ctx, s.renderer
	s.mu.Unlock()

	retryAfter, err := r.RenderEvent(ctx, eventID)

	switch {
	case err == nil && retryAfter > 0:
		s.log.WarnContext(
			ctx, "queue render rate limited",
			slog.Int64("event_id", eventID),
			slog.Duration("retry_after", retryAfter),
		)

		s.schedule(eventID, retryAfter)

	case err == nil:
		s.mu.Lock()
		delete(s.failures, eventID)
		s.mu.Unlock()

	case ctx.Err() != nil:

	default:
		s.retry(ctx, eventID, err)
	}
}

func (s *Scheduler) retry(ctx context.Context, eventID int64, err error) {
	s.mu.Lock()
	s.failures[eventID]++
	attempts := s.failures[eventID]

	giveUp := attempts >= s.cfg.MaxAttempts
	if giveUp {
		delete(s.failures, eventID)
	}

	s.mu.Unlock()

	if giveUp {
		s.log.ErrorContext(
			ctx, "queue render failed, giving up",
			slog.Int64("event_id", eventID),
			slog.Int("attempts", attempts),
			slog.Any("err", err),
		)

		return
	}

	s.log.WarnContext(
		ctx, "queue render failed, retrying",
		slog.Int64("event_id", eventID),
		slog.Int("attempts", attempts),
		slog.Any("err", err),
	)

	s.schedule(eventID, s.cfg.RetryDelay)
}
