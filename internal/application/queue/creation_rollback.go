package queue

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const eventCreationRollbackTimeout = 20 * time.Second

func (s *Service) rollbackCreatedEvent(ctx context.Context, event domain.Event) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), eventCreationRollbackTimeout)
	defer cancel()

	var rollbackErr error

	if event.MessageID != 0 {
		if err := s.messages.Delete(ctx, event); err != nil {
			rollbackErr = fmt.Errorf("delete published message: %w", err)
		}
	}

	if rollbackErr == nil {
		if err := s.events.Delete(ctx, event.ID); err != nil {
			rollbackErr = fmt.Errorf("delete failed event: %w", err)
		}
	}

	if rollbackErr != nil {
		s.log.ErrorContext(
			ctx, "event creation rollback incomplete",
			slog.Int64(AttrEventID, event.ID),
			slog.String(AttrTitle, event.Title),
			slog.Int64(AttrEventChatID, event.ChatID),
			slog.Int(AttrMessageID, event.MessageID),
			slog.Any("err", rollbackErr),
		)
	}

	return rollbackErr
}
