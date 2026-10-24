package queue

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

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
