package queue

import (
	"fmt"
	"strconv"
	"strings"
)

type ActionKind string

const (
	ActionJoin          ActionKind = "join"
	ActionLeave         ActionKind = "leave"
	ActionList          ActionKind = "list"
	ActionMenu          ActionKind = "menu"
	ActionClose         ActionKind = "close"
	ActionOpen          ActionKind = "open"
	ActionDelete        ActionKind = "delete"
	ActionDeleteConfirm ActionKind = "delconfirm"
	ActionKick          ActionKind = "kick"
	ActionSkip          ActionKind = "skip"
	ActionCarry         ActionKind = "carry"
	ActionCarryCancel   ActionKind = "carrycancel"
)

type Action struct {
	Kind    ActionKind
	EventID int64
	UserID  int64
}

func (a Action) String() string {
	if a.Kind == ActionKick {
		return fmt.Sprintf("%s:%d:%d", a.Kind, a.EventID, a.UserID)
	}

	return fmt.Sprintf("%s:%d", a.Kind, a.EventID)
}

func ParseAction(s string) (Action, error) {
	kind, rest, _ := strings.Cut(s, ":")
	rawEventID, rawUserID, hasUserID := strings.Cut(rest, ":")

	a := Action{Kind: ActionKind(kind)}

	if hasUserID != (a.Kind == ActionKick) {
		return Action{}, ErrUnknownAction
	}

	var err error

	if a.EventID, err = strconv.ParseInt(rawEventID, 10, 64); err != nil {
		return Action{}, fmt.Errorf("parse event id: %w", err)
	}

	if hasUserID {
		if a.UserID, err = strconv.ParseInt(rawUserID, 10, 64); err != nil {
			return Action{}, fmt.Errorf("parse user id: %w", err)
		}
	}

	return a, nil
}
