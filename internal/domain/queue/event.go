package queue

import (
	"strings"
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
}

func NewEvent(target Target, title string) (Event, error) {
	title, err := NormalizeTitle(title)
	if err != nil {
		return Event{}, err
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

func NormalizeTitle(title string) (string, error) {
	title = strings.TrimSpace(title)

	switch {
	case title == "":
		return "", ErrEmptyTitle

	case utf8.RuneCountInString(title) > MaxTitleLength:
		return "", ErrTitleTooLong
	}

	return title, nil
}
