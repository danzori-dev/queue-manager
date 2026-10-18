package queue

import (
	"fmt"
	"strconv"
	"strings"
)

type ActionKind string

const (
	ActionJoin  ActionKind = "join"
	ActionLeave ActionKind = "leave"
)

type Action struct {
	Kind    ActionKind
	EventID int64
}

func (a Action) String() string {
	return fmt.Sprintf("%s:%d", string(a.Kind), a.EventID)
}

func ParseAction(s string) (Action, error) {
	kind, rawID, ok := strings.Cut(s, ":")
	if !ok {
		return Action{}, ErrUnknownAction
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return Action{}, fmt.Errorf("parse event id: %w", err)
	}

	switch ActionKind(kind) {
	case ActionJoin, ActionLeave:
		return Action{
			Kind:    ActionKind(kind),
			EventID: id,
		}, nil

	default:
		return Action{}, ErrUnknownAction
	}
}
