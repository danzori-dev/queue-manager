package queue

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
	"github.com/danzori-dev/queue-manager/internal/domain/student"
)

func (s *Service) Join(ctx context.Context, eventID, userID int64, username string) (JoinResult, error) {
	if _, err := s.loadOpenEvent(ctx, eventID); err != nil {
		return JoinResult{}, err
	}

	added, err := s.entries.Add(ctx, eventID, userID, username)
	if err != nil {
		return JoinResult{}, fmt.Errorf("add entry: %w", err)
	}

	position, err := s.entries.Position(ctx, eventID, userID)
	if err != nil {
		return JoinResult{}, fmt.Errorf("get position: %w", err)
	}

	if added {
		s.scheduler.Schedule(eventID)
		s.audit(
			ctx, userID, ActionQueueJoined,
			slog.Int64(AttrEventID, eventID),
			slog.Int(AttrPosition, position),
		)
	}

	return JoinResult{
		Position: position,
		Already:  !added,
	}, nil
}

func (s *Service) Leave(ctx context.Context, eventID, userID int64) (bool, error) {
	if _, err := s.loadOpenEvent(ctx, eventID); err != nil {
		return false, err
	}

	removed, err := s.entries.Remove(ctx, eventID, userID)
	if err != nil {
		return false, fmt.Errorf("remove entry: %w", err)
	}

	if removed {
		s.scheduler.Schedule(eventID)
		s.audit(ctx, userID, ActionQueueLeft, slog.Int64(AttrEventID, eventID))
	}

	return removed, nil
}

func (s *Service) Skip(ctx context.Context, eventID, userID int64) (int, error) {
	if _, err := s.loadOpenEvent(ctx, eventID); err != nil {
		return 0, err
	}

	if err := s.entries.SwapWithNext(ctx, eventID, userID); err != nil {
		return 0, fmt.Errorf("swap with next: %w", err)
	}

	s.scheduler.Schedule(eventID)

	position, err := s.entries.Position(ctx, eventID, userID)
	if err != nil {
		return 0, fmt.Errorf("get position: %w", err)
	}

	s.audit(
		ctx, userID, ActionQueuePlaceSkipped,
		slog.Int64(AttrEventID, eventID),
		slog.Int(AttrPosition, position),
	)

	return position, nil
}

func (s *Service) ParticipantCandidates(ctx context.Context, eventID int64) ([]student.Student, error) {
	if _, err := s.events.Get(ctx, eventID); err != nil {
		return nil, err
	}

	entries, err := s.entries.List(ctx, eventID)
	if err != nil {
		return nil, err
	}

	candidates := s.students.List()

	candidates = slices.DeleteFunc(
		candidates, func(candidate student.Student) bool {
			return slices.ContainsFunc(
				entries, func(entry domain.Entry) bool {
					return entry.UserID == candidate.ID
				},
			)
		},
	)

	return candidates, nil
}

func (s *Service) AddParticipant(ctx context.Context, eventID, actorID, userID int64) (bool, error) {
	if _, err := s.events.Get(ctx, eventID); err != nil {
		return false, err
	}

	studentName, ok := s.students.Name(userID)
	if !ok {
		return false, student.ErrStudentNotFound
	}

	added, err := s.entries.Add(ctx, eventID, userID, "")
	if err != nil {
		return false, fmt.Errorf("add entry: %w", err)
	}

	if !added {
		return false, nil
	}

	s.scheduler.Schedule(eventID)
	s.audit(
		ctx, actorID, ActionParticipantAdded,
		slog.Int64(AttrEventID, eventID),
		slog.Int64(AttrActorID, actorID),
		slog.Int64(AttrPeerUserID, userID),
		slog.String(AttrPeerName, studentName),
	)

	return true, nil
}

func (s *Service) RemoveParticipant(ctx context.Context, eventID, actorID, userID int64) (bool, error) {
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return false, err
	}

	removed, err := s.entries.Remove(ctx, eventID, userID)
	if err != nil {
		return false, fmt.Errorf("remove entry: %w", err)
	}

	if !removed {
		return false, nil
	}

	s.scheduler.Schedule(eventID)

	if actorID == userID {
		s.audit(ctx, userID, ActionQueueLeft, slog.Int64(AttrEventID, eventID))

		return true, nil
	}

	userName := s.studentName(userID)

	s.audit(
		ctx, actorID, ActionParticipantRemoved,
		slog.Int64(AttrEventID, eventID),
		slog.Int64(AttrPeerUserID, userID),
		slog.String(AttrPeerName, userName),
	)

	actorName, _ := s.students.Name(actorID)

	s.messages.AnnounceRemoval(
		ctx, Removal{
			Event:     event,
			ActorName: actorName,
			UserID:    userID,
			UserName:  userName,
		},
	)

	return true, nil
}
