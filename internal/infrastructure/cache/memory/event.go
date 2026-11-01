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
	DeleteClosedBefore(ctx context.Context, before time.Time) ([]domain.Event, error)
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
	c.mu.Lock()
	defer c.mu.Unlock()

	id, err := c.inner.Create(ctx, event)
	if err != nil {
		return 0, err
	}

	event.ID = id
	c.events[id] = event

	return id, nil
}

func (c *EventCache) Get(ctx context.Context, eventID int64) (domain.Event, error) {
	c.mu.RLock()
	event, ok := c.events[eventID]
	c.mu.RUnlock()

	if ok {
		return event, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if cached, exists := c.events[eventID]; exists {
		return cached, nil
	}

	event, err := c.inner.Get(ctx, eventID)
	if err != nil {
		return domain.Event{}, err
	}

	c.events[eventID] = event

	return event, nil
}

func (c *EventCache) ListRecent(ctx context.Context, limit int) ([]domain.Event, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	events, err := c.inner.ListRecent(ctx, limit)
	if err != nil {
		return nil, err
	}

	for _, event := range events {
		c.events[event.ID] = event
	}

	return events, nil
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

	if err := c.inner.SetMessageID(ctx, eventID, messageID); err != nil {
		delete(c.events, eventID)

		return err
	}

	if event, ok := c.events[eventID]; ok {
		event.MessageID = messageID
		c.events[eventID] = event
	}

	return nil
}

func (c *EventCache) SetStatus(ctx context.Context, eventID int64, status domain.Status) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.inner.SetStatus(ctx, eventID, status); err != nil {
		delete(c.events, eventID)

		return err
	}

	if event, ok := c.events[eventID]; ok {
		event.Status = status
		c.events[eventID] = event
	}

	return nil
}

func (c *EventCache) SetTitle(ctx context.Context, eventID int64, title string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.inner.SetTitle(ctx, eventID, title); err != nil {
		delete(c.events, eventID)

		return err
	}

	if event, ok := c.events[eventID]; ok {
		event.Title = title
		c.events[eventID] = event
	}

	return nil
}

func (c *EventCache) Delete(ctx context.Context, eventID int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.inner.Delete(ctx, eventID)

	delete(c.events, eventID)

	return err
}

func (c *EventCache) DeleteClosedBefore(ctx context.Context, before time.Time) ([]domain.Event, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	deleted, err := c.inner.DeleteClosedBefore(ctx, before)
	if err != nil {
		clear(c.events)

		return nil, err
	}

	if len(deleted) > 0 {
		clear(c.events)
	}

	return deleted, nil
}
