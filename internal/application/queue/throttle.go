package queue

import (
	"slices"
	"sync"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type requestPair struct {
	fromID int64
	toID   int64
}

type RequestLimit struct {
	Window      time.Duration
	MaxFailures int
	Pause       time.Duration
}

type requestThrottle struct {
	limit    RequestLimit
	mu       sync.Mutex
	failures map[int64][]time.Time
	pauses   map[requestPair]time.Time
}

func newRequestThrottle(limit RequestLimit) *requestThrottle {
	return &requestThrottle{
		limit:    limit,
		failures: make(map[int64][]time.Time),
		pauses:   make(map[requestPair]time.Time),
	}
}

func (t *requestThrottle) check(fromID, toID int64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	recent := t.recent(fromID, now)

	if len(recent) >= t.limit.MaxFailures {
		freed := recent[len(recent)-t.limit.MaxFailures].Add(t.limit.Window)

		return &domain.RetryError{
			Err:   domain.ErrRequestsTooOften,
			After: freed.Sub(now),
		}
	}

	if until := t.pauses[requestPair{fromID: fromID, toID: toID}]; now.Before(until) {
		return &domain.RetryError{
			Err:   domain.ErrRequestCooldown,
			After: until.Sub(now),
		}
	}

	return nil
}

func (t *requestThrottle) fail(fromID, toID int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.prune(now)

	t.failures[fromID] = append(t.failures[fromID], now)
	t.pauses[requestPair{fromID: fromID, toID: toID}] = now.Add(t.limit.Pause)
}

func (t *requestThrottle) prune(now time.Time) {
	since := now.Add(-t.limit.Window)

	for fromID, times := range t.failures {
		times = slices.DeleteFunc(
			times, func(at time.Time) bool {
				return !at.After(since)
			},
		)

		if len(times) == 0 {
			delete(t.failures, fromID)

			continue
		}

		t.failures[fromID] = times
	}

	for pair, until := range t.pauses {
		if !now.Before(until) {
			delete(t.pauses, pair)
		}
	}
}

func (t *requestThrottle) recent(fromID int64, now time.Time) []time.Time {
	since := now.Add(-t.limit.Window)
	failures := t.failures[fromID]

	first := slices.IndexFunc(
		failures, func(at time.Time) bool {
			return at.After(since)
		},
	)
	if first < 0 {
		return nil
	}

	return failures[first:]
}
