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
	ListOpenWithUser(ctx context.Context, userID int64) ([]domain.Event, error)
	SetMessageID(ctx context.Context, eventID int64, messageID int) error
	SetStatus(ctx context.Context, eventID int64, status domain.Status) error
	SetTitle(ctx context.Context, eventID int64, title string) error
	Delete(ctx context.Context, eventID int64) error
	DeleteClosedBefore(ctx context.Context, before time.Time) (int, error)
}

type EntryRepository interface {
	Add(ctx context.Context, eventID, userID int64, username string) (bool, error)
	Remove(ctx context.Context, eventID, userID int64) (bool, error)
	Position(ctx context.Context, eventID, userID int64) (int, error)
	List(ctx context.Context, eventID int64) ([]domain.Entry, error)
	SwapWithNext(ctx context.Context, eventID, userID int64) error
	Prepend(ctx context.Context, eventID int64, entries []domain.Entry) error
	SwapSortKeys(ctx context.Context, eventID, userA, userB int64) error
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
	AskSwap(ctx context.Context, swap Swap) (int, error)
}

type RenderScheduler interface {
	Schedule(eventID int64)
}
