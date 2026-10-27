package activity

import (
	"context"
	"fmt"
	"slices"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/keyboard"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/reply"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/server/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

type Handler struct {
	reply    *reply.Replier
	journal  Journal
	students StudentService
}

func New(replier *reply.Replier, journal Journal, students StudentService) *Handler {
	return &Handler{
		reply:    replier,
		journal:  journal,
		students: students,
	}
}

func (h *Handler) Activity(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	kb, p := keyboard.ActivityPeople(h.people(), 0)

	return h.reply.SendMenu(ctx, msg, texts.ActivityPeople(p.Index, p.Count), kb)
}

func (h *Handler) Callback(ctx *th.Context, query telego.CallbackQuery) error {
	if !middleware.RoleFromContext(ctx).AtLeast(access.RoleOwner) {
		return h.reply.Answer(ctx, query, texts.AccessDenied)
	}

	action, err := callback.ParseAction(query.Data)
	if err != nil {
		return h.reply.Answer(ctx, query, "")
	}

	if err := h.show(ctx, query, action); err != nil {
		_ = h.reply.Answer(ctx, query, texts.InternalError)

		return err
	}

	return h.reply.Answer(ctx, query, "")
}

func (h *Handler) show(ctx context.Context, query telego.CallbackQuery, a callback.Action) error {
	switch a.Kind {
	case callback.ActionActivityList:
		kb, p := keyboard.ActivityPeople(h.people(), a.Page)

		return h.reply.EditMenu(ctx, query, texts.ActivityPeople(p.Index, p.Count), kb)

	case callback.ActionActivityLog:
		return h.showEvents(ctx, query, a.UserID, a.Page)

	default:
		return nil
	}
}

func (h *Handler) showEvents(ctx context.Context, query telego.CallbackQuery, userID int64, page int) error {
	index := slices.IndexFunc(
		h.people(), func(r keyboard.ActivityRow) bool {
			return r.UserID == userID
		},
	)

	events := h.journal.Events(userID)
	kb, p := keyboard.ActivityEvents(userID, keyboard.PageOf(max(index, 0)), len(events), page)
	start, end := p.Bounds()
	text := texts.ActivityEvents(h.name(userID), len(events), events[start:end], p.Index, p.Count)

	return h.reply.EditMenu(ctx, query, text, kb)
}

func (h *Handler) people() []keyboard.ActivityRow {
	students := h.students.List()
	rows := make([]keyboard.ActivityRow, 0, len(students))
	known := make(map[int64]bool, len(students))

	for _, s := range students {
		known[s.ID] = true
		rows = append(
			rows, keyboard.ActivityRow{
				UserID: s.ID,
				Name:   s.Name,
			},
		)
	}

	others := slices.DeleteFunc(
		h.journal.Users(), func(userID int64) bool {
			return known[userID]
		},
	)
	slices.Sort(others)

	for _, userID := range others {
		rows = append(
			rows, keyboard.ActivityRow{
				UserID: userID,
				Name:   unknownName(userID),
			},
		)
	}

	return rows
}

func (h *Handler) name(userID int64) string {
	if name, ok := h.students.Name(userID); ok {
		return name
	}

	return unknownName(userID)
}

func unknownName(userID int64) string {
	return fmt.Sprintf("id%d", userID)
}
