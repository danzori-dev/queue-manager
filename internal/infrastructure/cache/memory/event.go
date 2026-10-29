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
	SetTitle(ctx context.Context, eventID int64, title string) error
	Delete(ctx context.Context, eventID int64) error
	DeleteClosedBefore(ctx context.Context, before time.Time) (int, error)
}

type EventCache struct {
	inner EventRepository

	mu     sync.RWMutex
	events map[int64]domain.Event
}

func NewEventCache(inner EventRepository) *EventCache {
	return &EventCache{
		inner:  inner,
		events: make(map[int64]domain.Event),
	}
}

func (c *EventCache) Create(ctx context.Context, event domain.Event) (int64, error) {
	id, err := c.inner.Create(ctx, event)
	if err != nil {
		return 0, err
	}

	event.ID = id

	c.mu.Lock()
	c.events[id] = event
	c.mu.Unlock()

	return id, nil
}

func (c *EventCache) Get(ctx context.Context, eventID int64) (domain.Event, error) {
	c.mu.RLock()
	event, ok := c.events[eventID]
	c.mu.RUnlock()

	if ok {
		return event, nil
	}

	event, err := c.inner.Get(ctx, eventID)
	if err != nil {
		return domain.Event{}, err
	}

	c.mu.Lock()
	if cached, exists := c.events[eventID]; exists {
		c.mu.Unlock()

		return cached, nil
	}

	c.events[eventID] = event
	c.mu.Unlock()

	return event, nil
}

func (c *EventCache) ListRecent(ctx context.Context, limit int) ([]domain.Event, error) {
	events, err := c.inner.ListRecent(ctx, limit)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	for _, e := range events {
		c.events[e.ID] = e
	}

	c.mu.Unlock()

	return events, nil
}

func (c *EventCache) ListOpenIDs(ctx context.Context) ([]int64, error) {
	return c.inner.ListOpenIDs(ctx)
}

func (c *EventCache) ListOpenWithUser(ctx context.Context, userID int64) ([]domain.Event, error) {
	return c.inner.ListOpenWithUser(ctx, userID)
}

func (c *EventCache) SetMessageID(ctx context.Context, eventID int64, messageID int) error {
	if err := c.inner.SetMessageID(ctx, eventID, messageID); err != nil {
		return err
	}

	c.mu.Lock()
	if event, ok := c.events[eventID]; ok {
		event.MessageID = messageID
		c.events[eventID] = event
	}

	c.mu.Unlock()

	return nil
}

func (c *EventCache) SetStatus(ctx context.Context, eventID int64, status domain.Status) error {
	if err := c.inner.SetStatus(ctx, eventID, status); err != nil {
		return err
	}

	c.mu.Lock()
	if event, ok := c.events[eventID]; ok {
		event.Status = status
		c.events[eventID] = event
	}

	c.mu.Unlock()

	return nil
}

func (c *EventCache) SetTitle(ctx context.Context, eventID int64, title string) error {
	if err := c.inner.SetTitle(ctx, eventID, title); err != nil {
		return err
	}

	c.mu.Lock()
	if event, ok := c.events[eventID]; ok {
		event.Title = title
		c.events[eventID] = event
	}

	c.mu.Unlock()

	return nil
}

func (c *EventCache) Delete(ctx context.Context, eventID int64) error {
	c.mu.Lock()
	delete(c.events, eventID)
	c.mu.Unlock()

	return c.inner.Delete(ctx, eventID)
}

func (c *EventCache) DeleteClosedBefore(ctx context.Context, before time.Time) (int, error) {
	c.mu.Lock()
	clear(c.events)
	c.mu.Unlock()

	return c.inner.DeleteClosedBefore(ctx, before)
}
