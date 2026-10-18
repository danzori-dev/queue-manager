package queue

import "errors"

var (
	ErrEmptyTitle    = errors.New("event title is empty")
	ErrTitleTooLong  = errors.New("event title is too long")
	ErrTargetNotSet  = errors.New("queue target is not set")
	ErrEventNotFound = errors.New("event not found")
	ErrEventClosed   = errors.New("event is closed")
	ErrUnknownAction = errors.New("unknown queue action")
)
