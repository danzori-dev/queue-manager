package handler

import (
	"sync"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const confirmWindow = 5 * time.Second

type pressKey struct {
	kind     domain.ActionKind
	eventID  int64
	actorID  int64
	targetID int64
}

type confirmations struct {
	mu      sync.Mutex
	pressed map[pressKey]time.Time
}

func newConfirmations() *confirmations {
	return &confirmations{
		pressed: make(map[pressKey]time.Time),
	}
}

func (c *confirmations) Confirm(kind domain.ActionKind, eventID, actorID, targetID int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	for key, at := range c.pressed {
		if now.Sub(at) > confirmWindow {
			delete(c.pressed, key)
		}
	}

	key := pressKey{
		kind:     kind,
		eventID:  eventID,
		actorID:  actorID,
		targetID: targetID,
	}

	if _, ok := c.pressed[key]; ok {
		delete(c.pressed, key)

		return true
	}

	c.pressed[key] = now

	return false
}
