package queue

import (
	"context"
	"errors"
	"log/slog"
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

		delay := messageIDBackoff(attempt)

		s.log.InfoContext(
			ctx, "event message id save failed, retrying",
			slog.Int64(AttrEventID, eventID),
			slog.Int("message_id", messageID),
			slog.Int("attempt", attempt),
			slog.Duration("retry_delay", delay),
			slog.Any("err", err),
		)

		select {
		case <-ctx.Done():
			return errors.Join(err, ctx.Err())

		case <-time.After(delay):
		}
	}
}

func messageIDBackoff(attempt int) time.Duration {
	return messageIDSaveRetryDelay * time.Duration(1<<(attempt-1))
}
