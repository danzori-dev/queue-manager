package queue

import (
	"context"
	"fmt"
	"time"
)

func (s *Service) CleanupClosedEvents(ctx context.Context, olderThan time.Duration) (int, error) {
	deleted, err := s.events.DeleteClosedBefore(ctx, time.Now().Add(-olderThan))
	if err != nil {
		return 0, fmt.Errorf("delete closed events: %w", err)
	}

	return deleted, nil
}
