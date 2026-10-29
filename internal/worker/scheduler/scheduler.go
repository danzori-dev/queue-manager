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

type eventState struct {
	timer     *time.Timer
	inFlight  bool
	dirty     bool
	notBefore time.Time
	failures  int
}

type Scheduler struct {
	cfg Config
	log *slog.Logger

	mu       sync.Mutex
	ctx      context.Context //nolint:containedctx // scheduler lifetime ctx
	renderer Renderer
	states   map[int64]*eventState

	wg sync.WaitGroup
}

func New(cfg Config, log *slog.Logger) *Scheduler {
	return &Scheduler{
		cfg:    cfg,
		log:    log,
		states: make(map[int64]*eventState),
	}
}

func (s *Scheduler) Schedule(eventID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ctx == nil || s.ctx.Err() != nil {
		return
	}

	s.prune()

	st := s.state(eventID)
	if st.inFlight {
		st.dirty = true

		return
	}

	if st.timer != nil {
		return
	}

	s.armTimer(eventID, st, s.cfg.Debounce)
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
	for id, st := range s.states {
		if st.timer != nil && st.timer.Stop() {
			s.wg.Done()
		}

		delete(s.states, id)
	}
	s.mu.Unlock()

	s.wg.Wait()

	return nil
}

func (s *Scheduler) armTimer(eventID int64, st *eventState, delay time.Duration) {
	if wait := time.Until(st.notBefore); wait > delay {
		delay = wait
	}

	s.wg.Add(1)

	st.timer = time.AfterFunc(
		delay, func() {
			defer s.wg.Done()

			s.fire(eventID)
		},
	)
}

func (s *Scheduler) fire(eventID int64) {
	s.mu.Lock()
	st := s.state(eventID)
	st.timer = nil
	st.inFlight = true
	st.dirty = false
	ctx, r := s.ctx, s.renderer
	s.mu.Unlock()

	retryAfter, err := r.RenderEvent(ctx, eventID)

	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx.Err() != nil {
		return
	}

	st.inFlight = false
	now := time.Now()
	st.notBefore = now.Add(s.cfg.MinInterval)

	switch {
	case err == nil && retryAfter > 0:
		s.log.WarnContext(
			ctx, "queue render rate limited",
			slog.Int64("event_id", eventID),
			slog.Duration("retry_after", retryAfter),
		)

		st.notBefore = now.Add(retryAfter)
		s.armTimer(eventID, st, retryAfter)

	case err == nil:
		st.failures = 0

		if st.dirty {
			st.dirty = false
			s.armTimer(eventID, st, s.cfg.Debounce)
		}

	default:
		s.retry(ctx, eventID, st, err)
	}
}

func (s *Scheduler) retry(ctx context.Context, eventID int64, st *eventState, err error) {
	st.failures++
	attempts := st.failures

	if attempts >= s.cfg.MaxAttempts {
		delete(s.states, eventID)

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

	s.armTimer(eventID, st, s.cfg.RetryDelay)
}

func (s *Scheduler) state(eventID int64) *eventState {
	st, ok := s.states[eventID]
	if !ok {
		st = &eventState{}
		s.states[eventID] = st
	}

	return st
}

func (s *Scheduler) prune() {
	now := time.Now()

	for id, st := range s.states {
		if !st.inFlight && st.timer == nil && st.failures == 0 && !now.Before(st.notBefore) {
			delete(s.states, id)
		}
	}
}
