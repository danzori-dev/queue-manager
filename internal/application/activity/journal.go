package activity

import (
	"log/slog"
	"slices"
	"sync"
	"time"
)

const Retention = 24 * time.Hour

type Event struct {
	At     time.Time
	Action string
	Attrs  []slog.Attr
}

type Journal struct {
	mu     sync.Mutex
	people map[int64][]Event
}

func New() *Journal {
	return &Journal{
		people: make(map[int64][]Event),
	}
}

func (j *Journal) Record(userID int64, action string, attrs []slog.Attr) {
	j.mu.Lock()
	defer j.mu.Unlock()

	now := time.Now()

	j.people[userID] = append(
		j.recent(userID, now), Event{
			At:     now,
			Action: action,
			Attrs:  slices.Clone(attrs),
		},
	)
}

func (j *Journal) Users() []int64 {
	j.mu.Lock()
	defer j.mu.Unlock()

	now := time.Now()
	users := make([]int64, 0, len(j.people))

	for userID := range j.people {
		if len(j.recent(userID, now)) > 0 {
			users = append(users, userID)
		}
	}

	return users
}

func (j *Journal) Events(userID int64) []Event {
	j.mu.Lock()
	defer j.mu.Unlock()

	events := slices.Clone(j.recent(userID, time.Now()))
	slices.Reverse(events)

	return events
}

func (j *Journal) recent(userID int64, now time.Time) []Event {
	since := now.Add(-Retention)
	events := j.people[userID]

	first := slices.IndexFunc(
		events, func(e Event) bool {
			return e.At.After(since)
		},
	)
	if first < 0 {
		delete(j.people, userID)

		return nil
	}

	events = events[first:]
	j.people[userID] = events

	return events
}
