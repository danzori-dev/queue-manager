package queue

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
	"github.com/danzori-dev/queue-manager/internal/domain/student"
)

func (s *Service) InviteTargets(ctx context.Context, userID int64) ([]domain.Event, error) {
	events, err := s.events.ListOpenWithUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list open events with user: %w", err)
	}

	return events, nil
}

func (s *Service) InviteCandidates(ctx context.Context, eventID, userID int64) (InviteCandidates, error) {
	event, err := s.loadOpenEvent(ctx, eventID)
	if err != nil {
		return InviteCandidates{}, err
	}

	place, pos, err := s.placeOfUser(ctx, eventID, userID)
	if err != nil {
		return InviteCandidates{}, err
	}

	if len(place) >= domain.MaxPlaceSize {
		return InviteCandidates{}, domain.ErrPlaceFull
	}

	students := slices.DeleteFunc(
		s.students.List(), func(st student.Student) bool {
			return slices.ContainsFunc(
				place, func(e domain.Entry) bool {
					return e.UserID == st.ID
				},
			)
		},
	)

	return InviteCandidates{
		Event:    event,
		Position: pos,
		Students: students,
	}, nil
}

func (s *Service) RequestInvite(ctx context.Context, eventID, fromID, toID int64, fromMessageID int) (Invite, error) {
	if fromID == toID {
		return Invite{}, domain.ErrInviteSelf
	}

	event, err := s.loadOpenEvent(ctx, eventID)
	if err != nil {
		return Invite{}, err
	}

	places, err := s.places(ctx, eventID)
	if err != nil {
		return Invite{}, err
	}

	pos := domain.PlaceOf(places, fromID)
	if pos == 0 {
		return Invite{}, domain.ErrNotInQueue
	}

	toPos := domain.PlaceOf(places, toID)
	if toPos == pos {
		return Invite{}, domain.ErrAlreadyOnPlace
	}

	if len(places[pos-1]) >= domain.MaxPlaceSize {
		return Invite{}, domain.ErrPlaceFull
	}

	if err := s.throttle.check(fromID, toID); err != nil {
		return Invite{}, err
	}

	req, err := s.invites.add(
		domain.InviteRequest{
			Event:         event,
			FromID:        fromID,
			ToID:          toID,
			Position:      pos,
			ToPosition:    toPos,
			FromMessageID: fromMessageID,
			CreatedAt:     time.Now(),
		},
	)
	if err != nil {
		return Invite{}, err
	}

	invite := s.namedInvite(req)

	messageID, err := s.messages.AskInvite(ctx, invite)
	if err != nil {
		s.invites.remove(req.ID)

		return Invite{}, fmt.Errorf("ask invite: %w", err)
	}

	s.invites.setMessage(req.ID, messageID)
	invite.ToMessageID = messageID

	s.audit(
		ctx, fromID, ActionInviteSent,
		slog.Int64(AttrEventID, invite.Event.ID),
		slog.Int64(AttrPeerUserID, invite.ToID),
		slog.String(AttrPeerName, invite.ToName),
	)

	return invite, nil
}

func (s *Service) AcceptInvite(ctx context.Context, requestID, userID int64, username string) (Invite, error) {
	req, err := s.invites.take(requestID, userID, addressee)
	if err != nil {
		return Invite{}, err
	}

	invite := s.namedInvite(req)

	if _, err := s.loadOpenEvent(ctx, req.Event.ID); err != nil {
		if !errors.Is(err, domain.ErrEventClosed) && !errors.Is(err, domain.ErrEventNotFound) {
			s.invites.restore(req)
		}

		return invite, err
	}

	err = s.entries.JoinPlace(ctx, req.Event.ID, req.ToID, username, req.FromID)

	switch {
	case errors.Is(err, domain.ErrNotInQueue):
		return invite, domain.ErrInviteChanged

	case errors.Is(err, domain.ErrPlaceFull), errors.Is(err, domain.ErrAlreadyOnPlace):
		return invite, err

	case err != nil:
		s.invites.restore(req)

		return invite, fmt.Errorf("join place: %w", err)
	}

	s.scheduler.Schedule(req.Event.ID)

	pos, err := s.entries.Position(ctx, req.Event.ID, req.FromID)
	if err != nil {
		return invite, fmt.Errorf("get position: %w", err)
	}

	invite.Position = pos

	s.audit(
		ctx, userID, ActionInviteAccepted,
		slog.Int64(AttrEventID, invite.Event.ID),
		slog.Int64(AttrPeerUserID, invite.FromID),
		slog.String(AttrPeerName, invite.FromName),
		slog.Int(AttrPosition, pos),
	)

	return invite, nil
}

func (s *Service) DeclineInvite(ctx context.Context, requestID, userID int64) (Invite, error) {
	req, err := s.invites.take(requestID, userID, addressee)
	if err != nil {
		return Invite{}, err
	}

	s.throttle.fail(req.FromID, req.ToID)

	invite := s.namedInvite(req)
	s.audit(
		ctx, userID, ActionInviteDeclined,
		slog.Int64(AttrEventID, invite.Event.ID),
		slog.Int64(AttrPeerUserID, invite.FromID),
		slog.String(AttrPeerName, invite.FromName),
	)

	return invite, nil
}

func (s *Service) CancelInvite(ctx context.Context, requestID, userID int64) (Invite, error) {
	req, err := s.invites.take(requestID, userID, initiator)
	if err != nil {
		return Invite{}, err
	}

	s.throttle.fail(req.FromID, req.ToID)

	invite := s.namedInvite(req)
	s.audit(
		ctx, userID, ActionInviteCanceled,
		slog.Int64(AttrEventID, invite.Event.ID),
		slog.Int64(AttrPeerUserID, invite.ToID),
		slog.String(AttrPeerName, invite.ToName),
	)

	return invite, nil
}

func (s *Service) placeOfUser(ctx context.Context, eventID, userID int64) ([]domain.Entry, int, error) {
	places, err := s.places(ctx, eventID)
	if err != nil {
		return nil, 0, err
	}

	pos := domain.PlaceOf(places, userID)
	if pos == 0 {
		return nil, 0, domain.ErrNotInQueue
	}

	return places[pos-1], pos, nil
}

func (s *Service) places(ctx context.Context, eventID int64) ([][]domain.Entry, error) {
	entries, err := s.entries.List(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}

	return domain.Places(entries), nil
}

func (s *Service) namedInvite(req domain.InviteRequest) Invite {
	return Invite{
		InviteRequest: req,
		FromName:      s.studentName(req.FromID),
		ToName:        s.studentName(req.ToID),
	}
}
