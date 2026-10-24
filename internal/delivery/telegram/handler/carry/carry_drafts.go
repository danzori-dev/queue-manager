package carry

import (
	"sync"
	"time"
)

const carryDraftTTL = 5 * time.Minute

type carryDraft struct {
	fromID    int64
	fromPos   int
	toID      int64
	createdAt time.Time
}

type carryDrafts struct {
	mu     sync.Mutex
	drafts map[int64]carryDraft
}

func newCarryDrafts() *carryDrafts {
	return &carryDrafts{
		drafts: make(map[int64]carryDraft),
	}
}

func (d *carryDrafts) Put(adminID int64, draft carryDraft) {
	d.mu.Lock()
	defer d.mu.Unlock()

	draft.createdAt = time.Now()
	d.drafts[adminID] = draft
}

func (d *carryDrafts) Take(adminID, fromID int64) (carryDraft, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	draft, ok := d.drafts[adminID]
	if !ok || draft.fromID != fromID {
		return carryDraft{}, false
	}

	delete(d.drafts, adminID)

	if time.Since(draft.createdAt) > carryDraftTTL {
		return carryDraft{}, false
	}

	return draft, true
}
