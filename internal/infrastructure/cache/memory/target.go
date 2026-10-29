package memory

import (
	"context"
	"sync"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type TargetRepository interface {
	SetTarget(ctx context.Context, target domain.Target) error
	Target(ctx context.Context) (domain.Target, error)
}

type TargetCache struct {
	inner TargetRepository

	mu     sync.RWMutex
	target domain.Target
	loaded bool
}

func NewTargetCache(inner TargetRepository) *TargetCache {
	return &TargetCache{
		inner: inner,
	}
}

func (c *TargetCache) SetTarget(ctx context.Context, target domain.Target) error {
	if err := c.inner.SetTarget(ctx, target); err != nil {
		return err
	}

	c.mu.Lock()
	c.target = target
	c.loaded = true
	c.mu.Unlock()

	return nil
}

func (c *TargetCache) Target(ctx context.Context) (domain.Target, error) {
	c.mu.RLock()
	target, loaded := c.target, c.loaded
	c.mu.RUnlock()

	if loaded {
		return target, nil
	}

	target, err := c.inner.Target(ctx)
	if err != nil {
		return domain.Target{}, err
	}

	c.mu.Lock()
	if !c.loaded {
		c.target = target
		c.loaded = true
	}
	target = c.target
	c.mu.Unlock()

	return target, nil
}
