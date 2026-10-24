package carry

import (
	"context"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
)

type QueueService interface {
	PlanCarry(ctx context.Context, fromID int64, fromPos int, toID int64) (appqueue.CarryPlan, error)
	Carry(ctx context.Context, fromID int64, fromPos int, toID int64) (int, error)
}
