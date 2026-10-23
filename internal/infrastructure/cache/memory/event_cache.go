package memory

import (
	"context"
	"sync"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type EventRepository interface {
	Create(ctx context.Context, event domain.Event) (int64, error)
	Get(ctx context.Context, eventID int64) (domain.Event, error)
	ListRecent(ctx context.Context, limit int) ([]domain.Event, error)
	ListOpenIDs(ctx context.Context) ([]int64, error)
	ListOpenWithUser(ctx context.Context, userID int64) ([]domain.Event, error)
	SetMessageID(ctx context.Context, eventID int64, messageID int) error
	SetStatus(ctx context.Context, eventID int64, status domain.Status) error
	Delete(ctx context.Context, eventID int64) error
	DeleteClosedBefore(ctx context.Context, before time.Time) (int, error)
}

type EventCache struct {
	inner EventRepository

	mu     sync.Mutex
	events map[int64]domain.Event
}

func NewEventCache(inner EventRepository) *EventCache {
	return &EventCache{
		inner:  inner,
		events: make(map[int64]domain.Event),
	}
}

func (c *EventCache) Create(ctx context.Context, event domain.Event) (int64, error) {
	return c.inner.Create(ctx, event)
}

func (c *EventCache) Get(ctx context.Context, eventID int64) (domain.Event, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if event, ok := c.events[eventID]; ok {
		return event, nil
	}

	event, err := c.inner.Get(ctx, eventID)
	if err != nil {
		return domain.Event{}, err
	}

	c.events[eventID] = event

	return event, nil
}

func (c *EventCache) ListRecent(ctx context.Context, limit int) ([]domain.Event, error) {
	return c.inner.ListRecent(ctx, limit)
}

func (c *EventCache) ListOpenIDs(ctx context.Context) ([]int64, error) {
	return c.inner.ListOpenIDs(ctx)
}

func (c *EventCache) ListOpenWithUser(ctx context.Context, userID int64) ([]domain.Event, error) {
	return c.inner.ListOpenWithUser(ctx, userID)
}

func (c *EventCache) SetMessageID(ctx context.Context, eventID int64, messageID int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.events, eventID)

	return c.inner.SetMessageID(ctx, eventID, messageID)
}

func (c *EventCache) SetStatus(ctx context.Context, eventID int64, status domain.Status) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.events, eventID)

	return c.inner.SetStatus(ctx, eventID, status)
}

func (c *EventCache) Delete(ctx context.Context, eventID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.events, eventID)

	return c.inner.Delete(ctx, eventID)
}

func (c *EventCache) DeleteClosedBefore(ctx context.Context, before time.Time) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	clear(c.events)

	return c.inner.DeleteClosedBefore(ctx, before)
}
