package queue

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

func (s *Service) CleanupClosedEvents(ctx context.Context, olderThan time.Duration) (int, error) {
	deleted, err := s.events.DeleteClosedBefore(ctx, time.Now().Add(-olderThan))
	if err != nil {
		return 0, fmt.Errorf("delete closed events: %w", err)
	}

	for _, event := range deleted {
		s.log.InfoContext(
			ctx, "event deleted",
			slog.Int64("event_id", event.ID),
			slog.String("title", event.Title),
		)
	}

	return len(deleted), nil
}
