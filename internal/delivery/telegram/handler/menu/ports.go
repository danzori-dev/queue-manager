package menu

import (
	"context"

	"github.com/mymmrac/telego"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type QueueService interface {
	Events(ctx context.Context) ([]domain.Event, error)
	EventView(ctx context.Context, eventID int64) (appqueue.View, error)
	CloseEvent(ctx context.Context, eventID int64) error
	OpenEvent(ctx context.Context, eventID int64) error
	RemoveParticipant(ctx context.Context, eventID, actorID, userID int64) (bool, error)
	DeleteEvent(ctx context.Context, eventID int64) error
}

type CarryActions interface {
	Confirm(ctx context.Context, query telego.CallbackQuery, fromID int64) (string, error)
	Cancel(ctx context.Context, query telego.CallbackQuery, fromID int64) (string, error)
}
