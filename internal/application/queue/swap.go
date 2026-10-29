package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func (s *Service) SwapTargets(ctx context.Context, userID int64) ([]domain.Event, error) {
	events, err := s.events.ListOpenWithUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list open events with user: %w", err)
	}

	return events, nil
}

func (s *Service) SwapCandidates(ctx context.Context, eventID, userID int64) (View, error) {
	event, err := s.loadOpenEvent(ctx, eventID)
	if err != nil {
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

func (s *Service) RequestSwap(ctx context.Context, eventID, fromID, toID int64, fromMessageID int) (Swap, error) {
	if fromID == toID {
		return Swap{}, domain.ErrSwapSelf
	}

	event, err := s.loadOpenEvent(ctx, eventID)
	if err != nil {
		return Swap{}, err
	}

	fromPos, toPos, err := s.positions(ctx, eventID, fromID, toID)
	if err != nil {
		return Swap{}, err
	}

	if fromPos == toPos {
		return Swap{}, domain.ErrSwapSamePlace
	}

	if err := s.throttle.check(fromID, toID); err != nil {
		return Swap{}, err
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
		return Swap{}, err
	}

	swap := s.named(req)

	messageID, err := s.messages.AskSwap(ctx, swap)
	if err != nil {
		s.swaps.remove(req.ID)

		return Swap{}, fmt.Errorf("ask swap: %w", err)
	}

	s.swaps.setMessage(req.ID, messageID)
	swap.ToMessageID = messageID

	s.audit(
		ctx, fromID, ActionSwapRequested,
		slog.Int64(AttrEventID, swap.Event.ID),
		slog.Int64(AttrPeerUserID, swap.ToID),
		slog.String(AttrPeerName, swap.ToName),
	)

	return swap, nil
}

func (s *Service) AcceptSwap(ctx context.Context, requestID, userID int64) (Swap, error) {
	req, err := s.swaps.take(requestID, userID, addressee)
	if err != nil {
		return Swap{}, err
	}

	swap := s.named(req)

	if _, err := s.loadOpenEvent(ctx, req.Event.ID); err != nil {
		if !errors.Is(err, domain.ErrEventClosed) && !errors.Is(err, domain.ErrEventNotFound) {
			s.swaps.restore(req)
		}

		return swap, err
	}

	fromPos, toPos, err := s.positions(ctx, req.Event.ID, req.FromID, req.ToID)

	switch {
	case errors.Is(err, domain.ErrNotInQueue):
		return swap, domain.ErrSwapChanged

	case err != nil:
		s.swaps.restore(req)
		return swap, err

	case fromPos != req.FromPos || toPos != req.ToPos:
		return swap, domain.ErrSwapChanged
	}

	if err := s.entries.SwapSortKeys(ctx, req.Event.ID, req.FromID, req.ToID); err != nil {
		if errors.Is(err, domain.ErrNotInQueue) {
			return swap, domain.ErrSwapChanged
		}

		s.swaps.restore(req)
		return swap, fmt.Errorf("swap sort keys: %w", err)
	}

	s.scheduler.Schedule(req.Event.ID)
	s.audit(
		ctx, userID, ActionSwapAccepted,
		slog.Int64(AttrEventID, swap.Event.ID),
		slog.Int64(AttrPeerUserID, swap.FromID),
		slog.String(AttrPeerName, swap.FromName),
	)

	return swap, nil
}

func (s *Service) DeclineSwap(ctx context.Context, requestID, userID int64) (Swap, error) {
	req, err := s.swaps.take(requestID, userID, addressee)
	if err != nil {
		return Swap{}, err
	}

	s.throttle.fail(req.FromID, req.ToID)

	swap := s.named(req)
	s.audit(
		ctx, userID, ActionSwapDeclined,
		slog.Int64(AttrEventID, swap.Event.ID),
		slog.Int64(AttrPeerUserID, swap.FromID),
		slog.String(AttrPeerName, swap.FromName),
	)

	return swap, nil
}

func (s *Service) CancelSwap(ctx context.Context, requestID, userID int64) (Swap, error) {
	req, err := s.swaps.take(requestID, userID, initiator)
	if err != nil {
		return Swap{}, err
	}

	s.throttle.fail(req.FromID, req.ToID)

	swap := s.named(req)
	s.audit(
		ctx, userID, ActionSwapCanceled,
		slog.Int64(AttrEventID, swap.Event.ID),
		slog.Int64(AttrPeerUserID, swap.ToID),
		slog.String(AttrPeerName, swap.ToName),
	)

	return swap, nil
}

func (s *Service) positions(ctx context.Context, eventID, userA, userB int64) (posA, posB int, err error) {
	entries, err := s.entries.List(ctx, eventID)
	if err != nil {
		return 0, 0, fmt.Errorf("list entries: %w", err)
	}

	places := domain.Places(entries)
	posA = domain.PlaceOf(places, userA)
	posB = domain.PlaceOf(places, userB)

	if posA == 0 || posB == 0 {
		return 0, 0, domain.ErrNotInQueue
	}

	return posA, posB, nil
}

func (s *Service) named(req domain.SwapRequest) Swap {
	return Swap{
		SwapRequest: req,
		FromName:    s.studentName(req.FromID),
		ToName:      s.studentName(req.ToID),
	}
}
