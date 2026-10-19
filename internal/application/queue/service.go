package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const recentEventsLimit = 20

type Deps struct {
	Events    EventRepository
	Entries   EntryRepository
	Targets   TargetRepository
	Students  StudentDirectory
	Messages  Messenger
	Scheduler RenderScheduler
}

type Service struct {
	events    EventRepository
	entries   EntryRepository
	targets   TargetRepository
	students  StudentDirectory
	messages  Messenger
	scheduler RenderScheduler
}

func New(d Deps) *Service {
	return &Service{
		events:    d.Events,
		entries:   d.Entries,
		targets:   d.Targets,
		students:  d.Students,
		messages:  d.Messages,
		scheduler: d.Scheduler,
	}
}

type JoinResult struct {
	Position int
	Already  bool
}

func (s *Service) SetQueueTopic(ctx context.Context, chatID int64, threadID int) (bool, error) {
	_, err := s.targets.Target(ctx)
	if err != nil && !errors.Is(err, domain.ErrTargetNotSet) {
		return false, fmt.Errorf("get queue target: %w", err)
	}

	replaced := err == nil

	target := domain.Target{
		ChatID:   chatID,
		ThreadID: threadID,
	}

	if err := s.targets.SetTarget(ctx, target); err != nil {
		return false, fmt.Errorf("set queue target: %w", err)
	}

	return replaced, nil
}

func (s *Service) CreateEvent(ctx context.Context, title string) (domain.Event, error) {
	target, err := s.targets.Target(ctx)
	if err != nil {
		return domain.Event{}, fmt.Errorf("get queue target: %w", err)
	}

	event, err := domain.NewEvent(target, title)
	if err != nil {
		return domain.Event{}, err
	}

	event.ID, err = s.events.Create(ctx, event)
	if err != nil {
		return domain.Event{}, fmt.Errorf("create event: %w", err)
	}

	messageID, err := s.messages.Publish(ctx, View{Event: event})
	if err != nil {
		publishErr := fmt.Errorf("publish event: %w", err)

		if delErr := s.events.Delete(context.WithoutCancel(ctx), event.ID); delErr != nil {
			return domain.Event{}, errors.Join(publishErr, fmt.Errorf("delete unpublished event: %w", delErr))
		}

		return domain.Event{}, publishErr
	}

	if err := s.events.SetMessageID(ctx, event.ID, messageID); err != nil {
		return domain.Event{}, fmt.Errorf("save message id: %w", err)
	}

	event.MessageID = messageID

	return event, nil
}

func (s *Service) Join(ctx context.Context, eventID, userID int64, username string) (JoinResult, error) {
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return JoinResult{}, err
	}

	if err := event.CanJoin(); err != nil {
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
	}

	return JoinResult{
		Position: position,
		Already:  !added,
	}, nil
}

func (s *Service) Leave(ctx context.Context, eventID, userID int64) (bool, error) {
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return false, err
	}

	if err := event.CanJoin(); err != nil {
		return false, err
	}

	removed, err := s.entries.Remove(ctx, eventID, userID)
	if err != nil {
		return false, fmt.Errorf("remove entry: %w", err)
	}

	if removed {
		s.scheduler.Schedule(eventID)
	}

	return removed, nil
}

func (s *Service) Skip(ctx context.Context, eventID, userID int64) (int, error) {
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return 0, err
	}

	if err := event.CanJoin(); err != nil {
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

	return position, nil
}

func (s *Service) OpenEventIDs(ctx context.Context) ([]int64, error) {
	ids, err := s.events.ListOpenIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list open events: %w", err)
	}

	return ids, nil
}

func (s *Service) Events(ctx context.Context) ([]domain.Event, error) {
	events, err := s.events.ListRecent(ctx, recentEventsLimit)
	if err != nil {
		return nil, fmt.Errorf("list recent events: %w", err)
	}

	return events, nil
}

func (s *Service) EventView(ctx context.Context, eventID int64) (View, error) {
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return View{}, err
	}

	return s.view(ctx, event)
}

func (s *Service) CloseEvent(ctx context.Context, eventID int64) error {
	return s.setStatus(ctx, eventID, domain.StatusClosed)
}

func (s *Service) OpenEvent(ctx context.Context, eventID int64) error {
	return s.setStatus(ctx, eventID, domain.StatusOpen)
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

	actorName, _ := s.students.Name(actorID)

	s.messages.AnnounceRemoval(
		ctx, Removal{
			Event:     event,
			ActorName: actorName,
			UserID:    userID,
			UserName:  s.studentName(userID),
		},
	)

	return true, nil
}

func (s *Service) DeleteEvent(ctx context.Context, eventID int64) error {
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return err
	}

	if err := s.events.Delete(ctx, eventID); err != nil {
		return fmt.Errorf("delete event: %w", err)
	}

	s.messages.Delete(ctx, event)

	return nil
}

func (s *Service) RenderEvent(ctx context.Context, eventID int64) (time.Duration, error) {
	event, err := s.events.Get(ctx, eventID)

	switch {
	case errors.Is(err, domain.ErrEventNotFound):
		return 0, nil

	case err != nil:
		return 0, fmt.Errorf("get event: %w", err)
	}

	if event.MessageID == 0 {
		return 0, nil
	}

	view, err := s.view(ctx, event)
	if err != nil {
		return 0, err
	}

	retryAfter, err := s.messages.Edit(ctx, view)
	if errors.Is(err, domain.ErrMessageNotFound) {
		return 0, s.republish(ctx, view)
	}

	if err != nil {
		return 0, err
	}

	return retryAfter, nil
}

func (s *Service) view(ctx context.Context, event domain.Event) (View, error) {
	entries, err := s.entries.List(ctx, event.ID)
	if err != nil {
		return View{}, fmt.Errorf("list entries: %w", err)
	}

	lines := make([]Line, 0, len(entries))

	for i, e := range entries {
		lines = append(
			lines, Line{
				Position: i + 1,
				UserID:   e.UserID,
				Name:     s.studentName(e.UserID),
				UserName: e.Username,
			},
		)
	}

	return View{
		Event: event,
		Lines: lines,
	}, nil
}

func (s *Service) republish(ctx context.Context, view View) error {
	messageID, err := s.messages.Publish(ctx, view)
	if err != nil {
		return fmt.Errorf("republish event: %w", err)
	}

	if err := s.events.SetMessageID(ctx, view.Event.ID, messageID); err != nil {
		return fmt.Errorf("save message id: %w", err)
	}

	return nil
}

func (s *Service) setStatus(ctx context.Context, eventID int64, status domain.Status) error {
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return err
	}

	if event.Status == status {
		return nil
	}

	if err := s.events.SetStatus(ctx, eventID, status); err != nil {
		return fmt.Errorf("set event status: %w", err)
	}

	s.scheduler.Schedule(eventID)

	return nil
}

func (s *Service) studentName(userID int64) string {
	if name, ok := s.students.Name(userID); ok {
		return name
	}

	return fmt.Sprintf("id%d", userID)
}
