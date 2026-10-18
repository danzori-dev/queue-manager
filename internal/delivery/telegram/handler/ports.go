package handler

import (
	"context"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
	"github.com/danzori-dev/queue-manager/internal/domain/student"
)

type AdminService interface {
	Add(ctx context.Context, actorID, targetID int64) error
	Remove(ctx context.Context, actorID, targetID int64) error
	List(ctx context.Context, actorID int64) ([]int64, error)
}

type StudentService interface {
	Import(ctx context.Context, actorID int64, list map[int64]string) error
	List() []student.Student
	Name(userID int64) (string, bool)
	Count() int
}

type QueueService interface {
	SetQueueTopic(ctx context.Context, chatID int64, threadID int) (bool, error)
	CreateEvent(ctx context.Context, title string) (domain.Event, error)
	Join(ctx context.Context, eventID, userID int64, username string) (appqueue.JoinResult, error)
	Leave(ctx context.Context, eventID, userID int64) (bool, error)
}
