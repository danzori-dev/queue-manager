package menu

import (
	"context"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type QueueService interface {
	Events(ctx context.Context) ([]domain.Event, error)
	EventView(ctx context.Context, eventID int64) (appqueue.View, error)
	CloseEvent(ctx context.Context, eventID, actorID int64) error
	OpenEvent(ctx context.Context, eventID, actorID int64) error
	RenameEvent(ctx context.Context, eventID, actorID int64, name string) error
	RemoveParticipant(ctx context.Context, eventID, actorID, userID int64) (bool, error)
	DeleteEvent(ctx context.Context, eventID, actorID int64) error
}
