package queue

import "errors"

var (
	ErrEmptyTitle        = errors.New("event title is empty")
	ErrTitleTooLong      = errors.New("event title is too long")
	ErrTargetNotSet      = errors.New("queue target is not set")
	ErrEventNotFound     = errors.New("event not found")
	ErrEventClosed       = errors.New("event is closed")
	ErrUnknownAction     = errors.New("unknown queue action")
	ErrMessageNotFound   = errors.New("queue message not found")
	ErrNotInQueue        = errors.New("user is not in the queue")
	ErrLastInQueue       = errors.New("user is last in the queue")
	ErrCarrySameEvent    = errors.New("cannot carry over into the same event")
	ErrCarrySourceOpen   = errors.New("source event is still open")
	ErrCarryTargetClosed = errors.New("target event is closed")
	ErrCarryPosition     = errors.New("no such position in the source event")
)
