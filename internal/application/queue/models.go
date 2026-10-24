package queue

import domain "github.com/danzori-dev/queue-manager/internal/domain/queue"

type JoinResult struct {
	Position int
	Already  bool
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

type CarryPlan struct {
	From  domain.Event
	To    domain.Event
	Lines []Line
}

type Removal struct {
	Event     domain.Event
	ActorName string
	UserID    int64
	UserName  string
}

type Swap struct {
	domain.SwapRequest

	FromName string
	ToName   string
}
