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

	var lastErr error

	for attempt := 1; attempt <= messageIDSaveAttempts; attempt++ {
		lastErr := s.events.SetMessageID(ctx, eventID, messageID)

		if lastErr == nil || errors.Is(lastErr, domain.ErrEventNotFound) {
			return lastErr
		}

		if attempt == messageIDSaveAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return errors.Join(lastErr, ctx.Err())

		case <-time.After(messageIDBackoff(attempt)):
		}
	}

	return lastErr
}

func messageIDBackoff(attempt int) time.Duration {
	return messageIDSaveRetryDelay * time.Duration(1<<(attempt-1))
}
