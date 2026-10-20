package queue

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func (s *Service) SwapTargets(ctx context.Context, userID int64) ([]domain.Event, error) {
	ids, err := s.events.ListOpenIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list open events: %w", err)
	}

	var events []domain.Event

	for _, id := range ids {
		position, err := s.entries.Position(ctx, id, userID)
		if err != nil {
			return nil, fmt.Errorf("get position: %w", err)
		}

		if position == 0 {
			continue
		}

		event, err := s.events.Get(ctx, id)
		if errors.Is(err, domain.ErrEventNotFound) {
			continue
		}

		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

func (s *Service) SwapCandidates(ctx context.Context, eventID, userID int64) (View, error) {
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return View{}, err
	}

	if err := event.CanJoin(); err != nil {
		return View{}, err
	}

	view, err := s.view(ctx, event)
	if err != nil {
		return View{}, err
	}

	stands := slices.ContainsFunc(
		view.Lines, func(l Line) bool {
			return l.UserID == userID
		},
	)
	if !stands {
		return View{}, domain.ErrNotInQueue
	}

	return view, nil
}

func (s *Service) RequestSwap(ctx context.Context, eventID, fromID, toID int64, fromMessageID int) (domain.SwapRequest, error) {
	if fromID == toID {
		return domain.SwapRequest{}, domain.ErrSwapSelf
	}

	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return domain.SwapRequest{}, err
	}

	if err := event.CanJoin(); err != nil {
		return domain.SwapRequest{}, err
	}

	fromPos, toPos, err := s.positions(ctx, eventID, fromID, toID)
	if err != nil {
		return domain.SwapRequest{}, err
	}

	req, err := s.swaps.add(
		domain.SwapRequest{
			Event:         event,
			FromID:        fromID,
			ToID:          toID,
			FromPos:       fromPos,
			ToPos:         toPos,
			FromMessageID: fromMessageID,
			CreatedAt:     time.Now(),
		},
	)
	if err != nil {
		return domain.SwapRequest{}, err
	}

	ask := SwapAsk{
		Request:  req,
		FromName: s.studentName(fromID),
		ToName:   s.studentName(toID),
	}

	messageID, err := s.messages.AskSwap(ctx, ask)
	if err != nil {
		s.swaps.remove(req.ID)

		if errors.Is(err, domain.ErrSwapUnreachable) {
			return domain.SwapRequest{}, err
		}

		return domain.SwapRequest{}, fmt.Errorf("ask swap: %w", err)
	}

	s.swaps.setMessage(req.ID, messageID)
	req.ToMessageID = messageID

	return req, nil
}

func (s *Service) AcceptSwap(ctx context.Context, requestID, userID int64) (domain.SwapRequest, error) {
	req, err := s.swaps.take(requestID, userID, swapAddressee)
	if err != nil {
		return req, err
	}

	event, err := s.events.Get(ctx, req.Event.ID)
	if err != nil {
		return req, err
	}

	if err := event.CanJoin(); err != nil {
		return req, err
	}

	fromPos, toPos, err := s.positions(ctx, req.Event.ID, req.FromID, req.ToID)

	switch {
	case errors.Is(err, domain.ErrNotInQueue):
		return req, domain.ErrSwapChanged

	case err != nil:
		return req, err

	case fromPos != req.FromPos || toPos != req.ToPos:
		return req, domain.ErrSwapChanged
	}

	if err := s.entries.SwapSortKeys(ctx, req.Event.ID, req.FromID, req.ToID); err != nil {
		if errors.Is(err, domain.ErrNotInQueue) {
			return req, domain.ErrSwapChanged
		}

		return req, fmt.Errorf("swap sort keys: %w", err)
	}

	s.scheduler.Schedule(req.Event.ID)

	return req, nil
}

func (s *Service) positions(ctx context.Context, eventID, userA, userB int64) (posA, posB int, err error) {
	entries, err := s.entries.List(ctx, eventID)
	if err != nil {
		return 0, 0, fmt.Errorf("list entries: %w", err)
	}

	posA = slices.IndexFunc(
		entries, func(e domain.Entry) bool {
			return e.UserID == userA
		},
	) + 1
	posB = slices.IndexFunc(
		entries, func(e domain.Entry) bool {
			return e.UserID == userB
		},
	) + 1

	if posA == 0 || posB == 0 {
		return 0, 0, domain.ErrNotInQueue
	}

	return posA, posB, nil
}

func (s *Service) DeclineSwap(_ context.Context, requestID, userID int64) (domain.SwapRequest, error) {
	return s.swaps.take(requestID, userID, swapAddressee)
}

func (s *Service) CancelSwap(_ context.Context, requestID, userID int64) (domain.SwapRequest, error) {
	return s.swaps.take(requestID, userID, swapInitiator)
}
