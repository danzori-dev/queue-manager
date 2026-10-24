package menu

import (
	"context"
	"errors"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/confirm"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/keyboard"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/reply"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/server/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type Handler struct {
	reply    *reply.Replier
	queues   QueueService
	carries  CarryActions
	confirms *confirm.DoublePress
}

func New(replier *reply.Replier, queues QueueService, carries CarryActions) *Handler {
	return &Handler{
		reply:    replier,
		queues:   queues,
		carries:  carries,
		confirms: confirm.New(),
	}
}

func (h *Handler) Events(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleAdmin); !ok {
		return err
	}

	events, err := h.queues.Events(ctx)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return h.reply.SendHTML(ctx, msg, texts.EventsEmpty)
	}

	return h.reply.SendMenu(ctx, msg, texts.EventsTitle, keyboard.Events(events))
}

func (h *Handler) Callback(ctx *th.Context, query telego.CallbackQuery) error {
	if !middleware.RoleFromContext(ctx).AtLeast(access.RoleAdmin) {
		return h.reply.Answer(ctx, query, texts.AccessDenied)
	}

	action, err := callback.ParseAction(query.Data)
	if err != nil {
		return h.reply.Answer(ctx, query, "")
	}

	text, err := h.applyMenuAction(ctx, query, action)
	if err != nil {
		_ = h.reply.Answer(ctx, query, texts.InternalError)

		return err
	}

	return h.reply.Answer(ctx, query, text)
}

func (h *Handler) applyMenuAction(ctx context.Context, query telego.CallbackQuery, a callback.Action) (string, error) {
	switch a.Kind {
	case callback.ActionList:
		return "", h.showEvents(ctx, query)

	case callback.ActionMenu:
		return "", h.showMenu(ctx, query, a.EventID)

	case callback.ActionClose:
		return h.closeEvent(ctx, query, a.EventID)

	case callback.ActionOpen:
		return h.openEvent(ctx, query, a.EventID)

	case callback.ActionDelete:
		return "", h.showDeleteConfirm(ctx, query, a.EventID)

	case callback.ActionDeleteConfirm:
		return h.deleteEvent(ctx, query, a.EventID)

	case callback.ActionKick:
		return h.kick(ctx, query, a)

	case callback.ActionMembers:
		return "", h.showMembers(ctx, query, a.EventID, a.Page)

	case callback.ActionCarry:
		return h.carries.Confirm(ctx, query, a.EventID)

	case callback.ActionCarryCancel:
		return h.carries.Cancel(ctx, query, a.EventID)

	default:
		return "", callback.ErrUnknownAction
	}
}

func (h *Handler) showEvents(ctx context.Context, query telego.CallbackQuery) error {
	events, err := h.queues.Events(ctx)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return h.reply.EditMenu(ctx, query, texts.EventsEmpty, nil)
	}

	return h.reply.EditMenu(ctx, query, texts.EventsTitle, keyboard.Events(events))
}

func (h *Handler) showMenu(ctx context.Context, query telego.CallbackQuery, eventID int64) error {
	view, err := h.queues.EventView(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back())
	}

	if err != nil {
		return err
	}

	return h.reply.EditMenu(ctx, query, texts.EventMenu(view), keyboard.Menu(view))
}

func (h *Handler) closeEvent(ctx context.Context, query telego.CallbackQuery, eventID int64) (string, error) {
	err := h.queues.CloseEvent(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back())
	}

	if err != nil {
		return "", err
	}

	return texts.EventClosedAnswer, h.showMenu(ctx, query, eventID)
}

func (h *Handler) openEvent(ctx context.Context, query telego.CallbackQuery, eventID int64) (string, error) {
	err := h.queues.OpenEvent(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back())
	}

	if err != nil {
		return "", err
	}

	return texts.EventOpenedAnswer, h.showMenu(ctx, query, eventID)
}

func (h *Handler) showDeleteConfirm(ctx context.Context, query telego.CallbackQuery, eventID int64) error {
	view, err := h.queues.EventView(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back())
	}

	if err != nil {
		return err
	}

	return h.reply.EditMenu(ctx, query, texts.DeleteConfirm(view.Event.Title), keyboard.DeleteConfirm(eventID))
}

func (h *Handler) deleteEvent(ctx context.Context, query telego.CallbackQuery, eventID int64) (string, error) {
	err := h.queues.DeleteEvent(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back())
	}

	if err != nil {
		return "", err
	}

	return texts.EventDeleted, h.reply.EditMenu(ctx, query, texts.EventDeleted, keyboard.Back())
}

func (h *Handler) kick(ctx context.Context, query telego.CallbackQuery, a callback.Action) (string, error) {
	if !h.confirms.Confirm(callback.ActionKick, a.EventID, query.From.ID, a.UserID) {
		return texts.KickPressAgain, nil
	}

	removed, err := h.queues.RemoveParticipant(ctx, a.EventID, query.From.ID, a.UserID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back())
	}

	if err != nil {
		return "", err
	}

	text := texts.ParticipantRemoved

	if !removed {
		text = texts.ParticipantNotFound
	}

	return text, h.showMembers(ctx, query, a.EventID, a.Page)
}

func (h *Handler) showMembers(ctx context.Context, query telego.CallbackQuery, eventID int64, page int) error {
	view, err := h.queues.EventView(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back())
	}

	if err != nil {
		return err
	}

	kb, p := keyboard.Members(view, page, query.From.ID)

	return h.reply.EditMenu(ctx, query, texts.EventMembers(view, p.Index, p.Count), kb)
}
