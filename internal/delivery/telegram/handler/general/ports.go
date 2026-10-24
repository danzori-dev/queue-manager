package general

import (
	"context"

	"github.com/mymmrac/telego"
)

type QueueService interface {
	OpenEventIDs(ctx context.Context) ([]int64, error)
}

type SwapStarter interface {
	Start(ctx context.Context, msg telego.Message, eventID int64) error
}
