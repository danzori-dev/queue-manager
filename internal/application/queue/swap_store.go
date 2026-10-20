package queue

import (
	"sync"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type swapSide int

const (
	swapInitiator swapSide = iota
	swapAddressee
)

type swapStore struct {
	mu       sync.Mutex
	nextID   int64
	requests map[int64]domain.SwapRequest
}

func newSwapStore() *swapStore {
	return &swapStore{
		requests: make(map[int64]domain.SwapRequest),
	}
}

func (s *swapStore) add(req domain.SwapRequest) (domain.SwapRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, r := range s.requests {
		if r.Expired() {
			delete(s.requests, id)

			continue
		}

		if r.Involves(req.FromID) || r.Involves(req.ToID) {
			return domain.SwapRequest{}, domain.ErrSwapBusy
		}
	}

	s.nextID++
	req.ID = s.nextID
	s.requests[s.nextID] = req

	return req, nil
}

func (s *swapStore) setMessage(id int64, messageID int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req, ok := s.requests[id]; ok {
		req.ToMessageID = messageID
		s.requests[id] = req
	}
}

func (s *swapStore) remove(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.requests, id)
}

func (s *swapStore) take(id, userID int64, side swapSide) (domain.SwapRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req, ok := s.requests[id]
	if !ok {
		return domain.SwapRequest{}, domain.ErrSwapNotFound
	}

	allowed := req.FromID

	if side == swapAddressee {
		allowed = req.ToID
	}

	if allowed != userID {
		return domain.SwapRequest{}, domain.ErrSwapForbidden
	}

	delete(s.requests, id)

	if req.Expired() {
		return req, domain.ErrSwapExpired
	}

	return req, nil
}
