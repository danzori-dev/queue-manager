package queue

import (
	"context"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
)

type QueueService interface {
	SetQueueTopic(ctx context.Context, chatID int64, threadID int) (bool, error)
	CreateEvent(ctx context.Context, actorID int64, title string) error
	Join(ctx context.Context, eventID, userID int64, username string) (appqueue.JoinResult, error)
	Leave(ctx context.Context, eventID, userID int64) (bool, error)
	Skip(ctx context.Context, eventID, userID int64) (int, error)
}

type StudentService interface {
	Count() int
}
