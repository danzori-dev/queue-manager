package swap

import (
	"context"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type SwapService interface {
	SwapTargets(ctx context.Context, userID int64) ([]domain.Event, error)
	SwapCandidates(ctx context.Context, eventID, userID int64) (appqueue.View, error)
	RequestSwap(ctx context.Context, eventID, fromID, toID int64, fromMessageID int) (appqueue.Swap, error)
	AcceptSwap(ctx context.Context, requestID, userID int64) (appqueue.Swap, error)
	DeclineSwap(ctx context.Context, requestID, userID int64) (appqueue.Swap, error)
	CancelSwap(ctx context.Context, requestID, userID int64) (appqueue.Swap, error)
}
