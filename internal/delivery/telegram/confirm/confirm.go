package confirm

import (
	"sync"
	"time"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
)

const window = 5 * time.Second

type pressKey struct {
	kind     callback.ActionKind
	eventID  int64
	actorID  int64
	targetID int64
}

type DoublePress struct {
	mu      sync.Mutex
	pressed map[pressKey]time.Time
}

func New() *DoublePress {
	return &DoublePress{
		pressed: make(map[pressKey]time.Time),
	}
}

func (d *DoublePress) Confirm(kind callback.ActionKind, eventID, actorID, targetID int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	for key, at := range d.pressed {
		if now.Sub(at) > window {
			delete(d.pressed, key)
		}
	}

	key := pressKey{
		kind:     kind,
		eventID:  eventID,
		actorID:  actorID,
		targetID: targetID,
	}

	if _, ok := d.pressed[key]; ok {
		delete(d.pressed, key)

		return true
	}

	d.pressed[key] = now

	return false
}
