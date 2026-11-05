package queue

import (
	"sync"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const maxActiveInvites = 5

type storedInvite struct {
	request    domain.InviteRequest
	processing bool
}

type inviteStore struct {
	mu       sync.Mutex
	nextID   int64
	requests map[int64]storedInvite
}

func newInviteStore() *inviteStore {
	return &inviteStore{
		requests: make(map[int64]storedInvite),
	}
}

func (s *inviteStore) add(req domain.InviteRequest) (domain.InviteRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	activeInvites := 0

	for id, stored := range s.requests {
		r := stored.request

		if r.Expired() && !stored.processing {
			delete(s.requests, id)

			continue
		}

		if r.FromID == req.FromID {
			activeInvites++
		}

		if r.ToID == req.ToID {
			return domain.InviteRequest{}, domain.ErrInviteBusy
		}
	}

	if activeInvites >= maxActiveInvites {
		return domain.InviteRequest{}, domain.ErrTooManyInvites
	}

	s.nextID++
	req.ID = s.nextID
	s.requests[s.nextID] = storedInvite{
		request: req,
	}

	return req, nil
}

func (s *inviteStore) activeRecipients(fromID int64) []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	recipients := make([]int64, 0, maxActiveInvites)

	for _, stored := range s.requests {
		req := stored.request

		if req.FromID != fromID {
			continue
		}

		if req.Expired() && !stored.processing {
			continue
		}

		recipients = append(recipients, req.ToID)
	}

	return recipients
}

func (s *inviteStore) setMessage(id int64, messageID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if stored, ok := s.requests[id]; ok {
		stored.request.ToMessageID = messageID
		s.requests[id] = stored
	}
}

func (s *inviteStore) remove(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.requests, id)
}

func (s *inviteStore) begin(id, userID int64, messageID int, side side) (domain.InviteRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.requests[id]
	if !ok {
		return domain.InviteRequest{}, domain.ErrInviteNotFound
	}

	req := stored.request
	allowed := req.FromID
	expectedMessageID := req.FromMessageID

	if side == addressee {
		allowed = req.ToID
		expectedMessageID = req.ToMessageID
	}

	if allowed != userID {
		return domain.InviteRequest{}, domain.ErrInviteForbidden
	}

	if expectedMessageID == 0 {
		return domain.InviteRequest{}, domain.ErrInviteProcessing
	}

	if expectedMessageID != messageID {
		return domain.InviteRequest{}, domain.ErrInviteNotFound
	}

	if stored.processing {
		return domain.InviteRequest{}, domain.ErrInviteProcessing
	}

	if req.Expired() {
		delete(s.requests, id)

		return req, domain.ErrInviteExpired
	}

	stored.processing = true
	s.requests[id] = stored

	return req, nil
}

func (s *inviteStore) release(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored, ok := s.requests[id]
	if !ok {
		return
	}

	if stored.request.Expired() {
		delete(s.requests, id)

		return
	}

	stored.processing = false
	s.requests[id] = stored
}

func (s *inviteStore) take(id, userID int64, messageID int, side side) (domain.InviteRequest, error) {
	req, err := s.begin(id, userID, messageID, side)
	if err != nil {
		return domain.InviteRequest{}, err
	}

	s.remove(id)

	return req, nil
}
