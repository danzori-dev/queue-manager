package queue

import (
	"sync"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type inviteStore struct {
	mu       sync.Mutex
	nextID   int64
	requests map[int64]domain.InviteRequest
}

func newInviteStore() *inviteStore {
	return &inviteStore{
		requests: make(map[int64]domain.InviteRequest),
	}
}

func (s *inviteStore) add(req domain.InviteRequest) (domain.InviteRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, r := range s.requests {
		if r.Expired() {
			delete(s.requests, id)

			continue
		}

		if r.Involves(req.FromID) || r.Involves(req.ToID) {
			return domain.InviteRequest{}, domain.ErrInviteBusy
		}
	}

	s.nextID++
	req.ID = s.nextID
	s.requests[s.nextID] = req

	return req, nil
}

func (s *inviteStore) setMessage(id int64, messageID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req, ok := s.requests[id]; ok {
		req.ToMessageID = messageID
		s.requests[id] = req
	}
}

func (s *inviteStore) remove(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.requests, id)
}

func (s *inviteStore) restore(req domain.InviteRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !req.Expired() {
		s.requests[req.ID] = req
	}
}

func (s *inviteStore) take(id, userID int64, side side) (domain.InviteRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req, ok := s.requests[id]
	if !ok {
		return domain.InviteRequest{}, domain.ErrInviteNotFound
	}

	allowed := req.FromID

	if side == addressee {
		allowed = req.ToID
	}

	if allowed != userID {
		return domain.InviteRequest{}, domain.ErrInviteForbidden
	}

	delete(s.requests, id)

	if req.Expired() {
		return req, domain.ErrInviteExpired
	}

	return req, nil
}
