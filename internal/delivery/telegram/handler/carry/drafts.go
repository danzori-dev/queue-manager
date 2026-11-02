package carry

import (
	"errors"
	"sync"
	"time"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
)

const carryDraftTTL = 5 * time.Minute

var (
	errCarryExpired    = errors.New("carry draft expired")
	errCarryProcessing = errors.New("carry draft is being processed")
)

type carryDraft struct {
	fromID  int64
	fromPos int
	toID    int64

	targetsPage int
	sourcesPage int

	chatID         int64
	messageID      int
	confirmationID int64

	plan       *appqueue.CarryPlan
	createdAt  time.Time
	processing bool
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

func (d *carryDrafts) Start(adminID, chatID int64) (carryDraft, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if current, ok := d.drafts[adminID]; ok && current.processing {
		return carryDraft{}, errCarryProcessing
	}

	draft := carryDraft{
		chatID:     chatID,
		createdAt:  time.Now(),
		processing: true,
	}

	d.drafts[adminID] = draft

	return draft, nil
}

func (d *carryDrafts) Begin(adminID, chatID int64, messageID int, confirmationID int64) (carryDraft, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	draft, ok := d.drafts[adminID]
	if !ok {
		return carryDraft{}, errCarryExpired
	}

	if draft.chatID != chatID || draft.messageID != messageID {
		return carryDraft{}, errCarryExpired
	}

	if draft.processing {
		return carryDraft{}, errCarryProcessing
	}

	if time.Since(draft.createdAt) > carryDraftTTL {
		delete(d.drafts, adminID)

		return carryDraft{}, errCarryExpired
	}

	if confirmationID != 0 && draft.confirmationID != confirmationID {
		return carryDraft{}, errCarryExpired
	}

	draft.processing = true
	d.drafts[adminID] = draft

	return draft, nil
}

func (d *carryDrafts) Save(adminID int64, draft carryDraft) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if time.Since(draft.createdAt) > carryDraftTTL {
		delete(d.drafts, adminID)

		return
	}

	draft.processing = false
	d.drafts[adminID] = draft
}

func (d *carryDrafts) Delete(adminID int64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.drafts, adminID)
}

func (d *carryDraft) resetPlan() {
	d.toID = 0
	d.plan = nil
	d.confirmationID++
}

func (d *carryDraft) setPlan(plan appqueue.CarryPlan) {
	d.toID = plan.To.ID
	d.plan = &plan
	d.confirmationID++
}
