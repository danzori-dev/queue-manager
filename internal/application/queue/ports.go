package queue

import (
	"context"
	"time"

	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type EventRepository interface {
	Create(ctx context.Context, event domain.Event) (int64, error)
	Get(ctx context.Context, eventID int64) (domain.Event, error)
	ListRecent(ctx context.Context, limit int) ([]domain.Event, error)
	ListOpenIDs(ctx context.Context) ([]int64, error)
	SetMessageID(ctx context.Context, eventID int64, messageID int) error
	SetStatus(ctx context.Context, eventID int64, status domain.Status) error
	Delete(ctx context.Context, eventID int64) error
}

type EntryRepository interface {
	Add(ctx context.Context, eventID, userID int64, username string) (bool, error)
	Remove(ctx context.Context, eventID, userID int64) (bool, error)
	Position(ctx context.Context, eventID, userID int64) (int, error)
	List(ctx context.Context, eventID int64) ([]domain.Entry, error)
	SwapWithNext(ctx context.Context, eventID, userID int64) error
}

type TargetRepository interface {
	SetTarget(ctx context.Context, target domain.Target) error
	Target(ctx context.Context) (domain.Target, error)
}

type StudentDirectory interface {
	Name(userID int64) (string, bool)
}

type Messenger interface {
	Publish(ctx context.Context, view View) (int, error)
	Edit(ctx context.Context, view View) (time.Duration, error)
	Delete(ctx context.Context, event domain.Event)
	AnnounceRemoval(ctx context.Context, removal Removal)
}

type RenderScheduler interface {
	Schedule(eventID int64)
}

type View struct {
	Event domain.Event
	Lines []Line
}

type Line struct {
	Position int
	UserID   int64
	Name     string
	UserName string
}

type Removal struct {
	Event     domain.Event
	ActorName string
	UserID    int64
	UserName  string
}
