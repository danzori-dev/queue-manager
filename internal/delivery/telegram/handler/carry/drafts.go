package carry

import (
	"sync"
	"time"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
)

const carryDraftTTL = 5 * time.Minute

type carryDraft struct {
	fromID      int64
	fromPos     int
	toID        int64
	targetsPage int
	sourcesPage int
	plan        *appqueue.CarryPlan
	createdAt   time.Time
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

func (d *carryDrafts) Get(adminID int64) (carryDraft, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	draft, ok := d.drafts[adminID]
	if !ok {
		return carryDraft{}, false
	}

	if time.Since(draft.createdAt) > carryDraftTTL {
		delete(d.drafts, adminID)

		return carryDraft{}, false
	}

	return draft, true
}

func (d *carryDrafts) Put(adminID int64, draft carryDraft) {
	d.mu.Lock()
	defer d.mu.Unlock()

	draft.createdAt = time.Now()
	d.drafts[adminID] = draft
}

func (d *carryDrafts) SetSourcesPage(adminID int64, page int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	draft, ok := d.drafts[adminID]
	if !ok {
		return false
	}

	if time.Since(draft.createdAt) > carryDraftTTL {
		delete(d.drafts, adminID)

		return false
	}

	draft.sourcesPage = page
	d.drafts[adminID] = draft

	return true
}

func (d *carryDrafts) SetPlan(adminID int64, fromPos int, plan appqueue.CarryPlan) (carryDraft, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	draft, ok := d.drafts[adminID]
	if !ok {
		return carryDraft{}, false
	}

	if time.Since(draft.createdAt) > carryDraftTTL {
		delete(d.drafts, adminID)

		return carryDraft{}, false
	}

	if draft.fromID != plan.From.ID || draft.fromPos != fromPos {
		return carryDraft{}, false
	}

	if len(plan.Lines) == 0 {
		return carryDraft{}, false
	}

	draft.toID = plan.To.ID
	draft.plan = &plan
	d.drafts[adminID] = draft

	return draft, true
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

func (d *carryDrafts) TakePlan(adminID, fromID int64) (carryDraft, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	draft, ok := d.drafts[adminID]
	if !ok || draft.fromID != fromID {
		return carryDraft{}, false
	}

	if time.Since(draft.createdAt) > carryDraftTTL {
		delete(d.drafts, adminID)

		return carryDraft{}, false
	}

	if draft.plan == nil {
		return carryDraft{}, false
	}

	if draft.fromPos < 1 || draft.toID == 0 {
		return carryDraft{}, false
	}

	delete(d.drafts, adminID)

	return draft, true
}

func (d *carryDrafts) SelectSource(adminID, fromID int64) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	draft, ok := d.drafts[adminID]
	if !ok {
		return false
	}

	if time.Since(draft.createdAt) > carryDraftTTL {
		delete(d.drafts, adminID)

		return false
	}

	if draft.fromID != fromID {
		draft.fromID = fromID
		draft.fromPos = 0
		draft.toID = 0
		draft.targetsPage = 0
		draft.plan = nil

		d.drafts[adminID] = draft
	}

	return true
}

func (d *carryDrafts) SelectPlace(adminID, fromID int64, fromPos, targetsPage int) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	draft, ok := d.drafts[adminID]
	if !ok {
		return false
	}

	if time.Since(draft.createdAt) > carryDraftTTL {
		delete(d.drafts, adminID)

		return false
	}

	if draft.fromID != fromID {
		return false
	}

	if draft.fromPos != fromPos {
		draft.fromPos = fromPos
		draft.toID = 0
		draft.plan = nil
	}

	draft.targetsPage = targetsPage
	d.drafts[adminID] = draft

	return true
}
