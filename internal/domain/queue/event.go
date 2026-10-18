package queue

import (
	"strings"
	"time"
	"unicode/utf8"
)

const MaxTitleLength = 100

type Status string

const (
	StatusOpen   Status = "open"
	StatusClosed Status = "closed"
)

type Event struct {
	ID        int64
	ChatID    int64
	ThreadID  int
	MessageID int
	Title     string
	Status    Status
	CreatedAt time.Time
	ClosedAt  *time.Time
}

func NewEvent(target Target, title string) (Event, error) {
	title = strings.TrimSpace(title)

	switch {
	case title == "":
		return Event{}, ErrEmptyTitle

	case utf8.RuneCountInString(title) > MaxTitleLength:
		return Event{}, ErrTitleTooLong
	}

	return Event{
		ChatID:   target.ChatID,
		ThreadID: target.ThreadID,
		Title:    title,
		Status:   StatusOpen,
	}, nil
}

func (e Event) CanJoin() error {
	if e.Status != StatusOpen {
		return ErrEventClosed
	}

	return nil
}
