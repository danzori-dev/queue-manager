package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

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

func (s *Service) RenderEvent(ctx context.Context, eventID int64) (time.Duration, error) {
	event, err := s.events.Get(ctx, eventID)
	if err != nil {
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
	if err != nil {
		return 0, err
	}

	return retryAfter, nil
}

func (s *Service) OpenEventIDs(ctx context.Context) ([]int64, error) {
	ids, err := s.events.ListOpenIDs(ctx)
	if err != nil {
		return nil, fmt.Errorf("list open events: %w", err)
	}

	return ids, nil
}

func (s *Service) view(ctx context.Context, event domain.Event) (View, error) {
	entries, err := s.entries.List(ctx, event.ID)
	if err != nil {
		return View{}, fmt.Errorf("list entries: %w", err)
	}

	lines := make([]Line, 0, len(entries))

	for i, e := range entries {
		name, ok := s.students.Name(e.UserID)
		if !ok {
			name = fmt.Sprintf("id%d", e.UserID)
		}

		lines = append(
			lines, Line{
				Position: i + 1,
				Name:     name,
				Username: e.Username,
			},
		)
	}

	return View{
		Event: event,
		Lines: lines,
	}, nil
}
