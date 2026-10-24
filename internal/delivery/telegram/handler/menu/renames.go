package menu

import (
	"sync"
	"time"
)

const renameTTL = 5 * time.Minute

type pendingRename struct {
	eventID   int64
	createdAt time.Time
}

type renames struct {
	mu      sync.Mutex
	pending map[int64]pendingRename
}

func newRenames() *renames {
	return &renames{
		pending: make(map[int64]pendingRename),
	}
}

func (r *renames) Put(adminID, eventID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pending[adminID] = pendingRename{
		eventID:   eventID,
		createdAt: time.Now(),
	}
}

func (r *renames) Take(adminID int64) (int64, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.pending[adminID]
	if !ok {
		return 0, false
	}

	delete(r.pending, adminID)

	if time.Since(p.createdAt) > renameTTL {
		return 0, false
	}

	return p.eventID, true
}

func (r *renames) Drop(adminID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.pending, adminID)
}
