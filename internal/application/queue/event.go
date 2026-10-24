package queue

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const recentEventsLimit = 20

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

func (s *Service) CreateEvent(ctx context.Context, title string) error {
	target, err := s.targets.Target(ctx)
	if err != nil {
		return fmt.Errorf("get queue target: %w", err)
	}

	event, err := domain.NewEvent(target, title)
	if err != nil {
		return err
	}

	event.ID, err = s.events.Create(ctx, event)
	if err != nil {
		return fmt.Errorf("create event: %w", err)
	}

	messageID, err := s.messages.Publish(ctx, View{Event: event})
	if err != nil {
		publishErr := fmt.Errorf("publish event: %w", err)

		if delErr := s.events.Delete(context.WithoutCancel(ctx), event.ID); delErr != nil {
			return errors.Join(publishErr, fmt.Errorf("delete unpublished event: %w", delErr))
		}

		return publishErr
	}

	if err := s.events.SetMessageID(ctx, event.ID, messageID); err != nil {
		return fmt.Errorf("save message id: %w", err)
	}

	return nil
}

func (s *Service) RenameEvent(ctx context.Context, eventID int64, title string) error {
	title, err := domain.NormalizeTitle(title)
	if err != nil {
		return err
	}

	event, err := s.events.Get(ctx, eventID)
	if err != nil {
		return err
	}

	if event.Title == title {
		return nil
	}

	if err := s.events.SetTitle(ctx, eventID, title); err != nil {
		return fmt.Errorf("set event title: %w", err)
	}

	s.scheduler.Schedule(eventID)

	return nil
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
