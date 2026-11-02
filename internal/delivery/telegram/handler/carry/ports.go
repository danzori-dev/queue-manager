package carry

import (
	"context"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type QueueService interface {
	CarrySources(ctx context.Context) ([]domain.Event, error)
	CarrySource(ctx context.Context, fromID int64) (appqueue.View, error)
	CarryTargets(ctx context.Context, fromID int64, fromPos int) ([]domain.Event, error)
	PlanCarry(ctx context.Context, fromID int64, fromPos int, toID int64) (appqueue.CarryPlan, error)
	Carry(ctx context.Context, fromID int64, fromPos int, toID, actorID int64) (int, error)
}
