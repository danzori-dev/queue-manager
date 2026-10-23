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
	ActionMembers       ActionKind = "members"
	ActionSkip          ActionKind = "skip"
	ActionCarry         ActionKind = "carry"
	ActionCarryCancel   ActionKind = "carrycancel"
	ActionSwapEvent     ActionKind = "swapevent"
	ActionSwapPick      ActionKind = "swappick"
	ActionSwapPage      ActionKind = "swappage"
	ActionSwapSend      ActionKind = "swapsend"
	ActionSwapAbort     ActionKind = "swapabort"
	ActionSwapAccept    ActionKind = "swapaccept"
	ActionSwapDecline   ActionKind = "swapdecline"
	ActionSwapCancel    ActionKind = "swapcancel"
)

type Action struct {
	Kind    ActionKind
	EventID int64
	UserID  int64
	Page    int
}

func (a Action) String() string {
	fields := []int64{a.EventID, a.UserID, int64(a.Page)}

	for len(fields) > 1 && fields[len(fields)-1] == 0 {
		fields = fields[:len(fields)-1]
	}

	var b strings.Builder

	b.WriteString(string(a.Kind))

	for _, f := range fields {
		b.WriteString(":" + strconv.FormatInt(f, 10))
	}

	return b.String()
}

func ParseAction(s string) (Action, error) {
	kind, rest, _ := strings.Cut(s, ":")
	raw := strings.Split(rest, ":")

	if kind == "" || len(raw) > 3 {
		return Action{}, ErrUnknownAction
	}

	var fields [3]int64

	for i, r := range raw {
		v, err := strconv.ParseInt(r, 10, 64)
		if err != nil {
			return Action{}, fmt.Errorf("parse action field %d: %w", i+1, err)
		}

		fields[i] = v
	}

	return Action{
		Kind:    ActionKind(kind),
		EventID: fields[0],
		UserID:  fields[1],
		Page:    int(fields[2]),
	}, nil
}
