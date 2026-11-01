package queue

import (
	"context"
	"errors"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const (
	messageIDSaveAttempts   = 3
	messageIDSaveRetryDelay = 500 * time.Millisecond
	messageIDSaveTimeout    = 20 * time.Second
)

func (s *Service) saveMessageID(ctx context.Context, eventID int64, messageID int) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), messageIDSaveTimeout)
	defer cancel()

	for attempt := 1; ; attempt++ {
		err := s.events.SetMessageID(ctx, eventID, messageID)

		if err == nil || errors.Is(err, domain.ErrEventNotFound) || attempt >= messageIDSaveAttempts {
			return err
		}

		select {
		case <-ctx.Done():
			return errors.Join(err, ctx.Err())

		case <-time.After(messageIDBackoff(attempt)):
		}
	}
}

func messageIDBackoff(attempt int) time.Duration {
	return messageIDSaveRetryDelay * time.Duration(1<<(attempt-1))
}
