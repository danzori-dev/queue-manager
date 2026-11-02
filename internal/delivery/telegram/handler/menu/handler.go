package menu

import (
	"context"
	"errors"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/command"
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
	renames  *renames
}

func New(replier *reply.Replier, queues QueueService, carries CarryActions) *Handler {
	return &Handler{
		reply:    replier,
		queues:   queues,
		carries:  carries,
		confirms: confirm.New(),
		renames:  newRenames(),
	}
}

func (h *Handler) Events(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleAdmin); !ok {
		return err
	}

	h.renames.Drop(msg.From.ID)

	events, err := h.queues.Events(ctx)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return h.reply.SendHTML(ctx, msg, texts.EventsEmpty)
	}

	kb, p := keyboard.Events(events, 0)

	return h.reply.SendMenu(ctx, msg, texts.EventsTitle(p.Index, p.Count), kb)
}

func (h *Handler) Callback(ctx *th.Context, query telego.CallbackQuery) error {
	if !middleware.RoleFromContext(ctx).AtLeast(access.RoleAdmin) {
		return h.reply.Answer(ctx, query, texts.AccessDenied)
	}

	action, err := callback.ParseAction(query.Data)
	if err != nil {
		return h.reply.Answer(ctx, query, "")
	}

	if action.Kind != callback.ActionRename {
		h.renames.Drop(query.From.ID)
	}

	text, err := h.applyMenuAction(ctx, query, action)
	if err != nil {
		_ = h.reply.Answer(ctx, query, texts.InternalError)

		return err
	}

	return h.reply.Answer(ctx, query, text)
}

func (h *Handler) Rename(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleAdmin); !ok {
		return err
	}

	eventID, title, ok := command.IDAndText(msg.Text)
	if !ok {
		return h.reply.SendHTML(ctx, msg, texts.RenameUsage)
	}

	h.renames.Drop(msg.From.ID)

	_, err := h.rename(ctx, msg, eventID, title, 0)

	return err
}

func (h *Handler) Text(ctx *th.Context, msg telego.Message) error {
	if !middleware.RoleFromContext(ctx).AtLeast(access.RoleAdmin) {
		return nil
	}

	eventID, eventsPage, ok := h.renames.Take(msg.From.ID)
	if !ok {
		return nil
	}

	renamed, err := h.rename(ctx, msg, eventID, msg.Text, eventsPage)
	if err != nil || !renamed {
		return err
	}

	view, err := h.queues.EventView(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return h.reply.SendMenu(ctx, msg, texts.EventGone, keyboard.Back(eventsPage))
	}

	if err != nil {
		return err
	}

	return h.reply.SendMenu(ctx, msg, texts.EventMenu(view), keyboard.Menu(view, eventsPage))
}

func (h *Handler) applyMenuAction(ctx context.Context, query telego.CallbackQuery, a callback.Action) (string, error) {
	switch a.Kind {
	case callback.ActionList:
		return "", h.showEvents(ctx, query, a.Page)

	case callback.ActionMenu, callback.ActionRenameCancel:
		return "", h.showMenu(ctx, query, a.EventID, a.EventsPage)

	case callback.ActionClose:
		return h.closeEvent(ctx, query, a.EventID, a.EventsPage)

	case callback.ActionOpen:
		return h.openEvent(ctx, query, a.EventID, a.EventsPage)

	case callback.ActionRename:
		return "", h.askRename(ctx, query, a.EventID, a.EventsPage)

	case callback.ActionDelete:
		return "", h.showDeleteConfirm(ctx, query, a.EventID, a.EventsPage)

	case callback.ActionDeleteConfirm:
		return h.deleteEvent(ctx, query, a.EventID, a.EventsPage)

	case callback.ActionKick:
		return h.kick(ctx, query, a)

	case callback.ActionMembers:
		return "", h.showMembers(ctx, query, a.EventID, a.Page, a.EventsPage)

	case callback.ActionCarry:
		return h.carries.Confirm(ctx, query, a.EventID)

	case callback.ActionCarryCancel:
		return h.carries.Cancel(ctx, query, a.EventID)

	default:
		return "", callback.ErrUnknownAction
	}
}

func (h *Handler) showEvents(ctx context.Context, query telego.CallbackQuery, page int) error {
	events, err := h.queues.Events(ctx)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return h.reply.EditMenu(ctx, query, texts.EventsEmpty, keyboard.None())
	}

	kb, p := keyboard.Events(events, page)

	return h.reply.EditMenu(ctx, query, texts.EventsTitle(p.Index, p.Count), kb)
}

func (h *Handler) showMenu(ctx context.Context, query telego.CallbackQuery, eventID int64, eventsPage int) error {
	view, err := h.queues.EventView(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back(eventsPage))
	}

	if err != nil {
		return err
	}

	return h.reply.EditMenu(ctx, query, texts.EventMenu(view), keyboard.Menu(view, eventsPage))
}

func (h *Handler) closeEvent(ctx context.Context, query telego.CallbackQuery, eventID int64, eventsPage int) (string, error) {
	err := h.queues.CloseEvent(ctx, eventID, query.From.ID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back(eventsPage))
	}

	if err != nil {
		return "", err
	}

	return texts.EventClosedAnswer, h.showMenu(ctx, query, eventID, eventsPage)
}

func (h *Handler) openEvent(ctx context.Context, query telego.CallbackQuery, eventID int64, eventsPage int) (string, error) {
	err := h.queues.OpenEvent(ctx, eventID, query.From.ID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back(eventsPage))
	}

	if err != nil {
		return "", err
	}

	return texts.EventOpenedAnswer, h.showMenu(ctx, query, eventID, eventsPage)
}

func (h *Handler) rename(ctx context.Context, msg telego.Message, eventID int64, title string, eventsPage int) (bool, error) {
	err := h.queues.RenameEvent(ctx, eventID, msg.From.ID, title)

	switch {
	case errors.Is(err, domain.ErrEmptyTitle):
		h.renames.Put(msg.From.ID, eventID, eventsPage)

		return false, h.reply.SendHTML(ctx, msg, texts.TitleEmpty)

	case errors.Is(err, domain.ErrTitleTooLong):
		h.renames.Put(msg.From.ID, eventID, eventsPage)

		return false, h.reply.SendHTML(ctx, msg, texts.EventTitleTooLong)

	case errors.Is(err, domain.ErrEventNotFound):
		return false, h.reply.SendMenu(ctx, msg, texts.EventGone, keyboard.Back(eventsPage))

	case err != nil:
		return false, err
	}

	return true, h.reply.SendHTML(ctx, msg, texts.EventRenamed)
}

func (h *Handler) askRename(ctx context.Context, query telego.CallbackQuery, eventID int64, eventsPage int) error {
	view, err := h.queues.EventView(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back(eventsPage))
	}

	if err != nil {
		return err
	}

	h.renames.Put(query.From.ID, eventID, eventsPage)

	return h.reply.EditMenu(ctx, query, texts.RenamePrompt(view.Event.Title, renameTTL), keyboard.RenameCancel(eventID, eventsPage))
}

func (h *Handler) showDeleteConfirm(ctx context.Context, query telego.CallbackQuery, eventID int64, eventsPage int) error {
	view, err := h.queues.EventView(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back(eventsPage))
	}

	if err != nil {
		return err
	}

	return h.reply.EditMenu(ctx, query, texts.DeleteConfirm(view.Event.Title), keyboard.DeleteConfirm(eventID, eventsPage))
}

func (h *Handler) deleteEvent(ctx context.Context, query telego.CallbackQuery, eventID int64, eventsPage int) (string, error) {
	err := h.queues.DeleteEvent(ctx, eventID, query.From.ID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back(eventsPage))
	}

	if err != nil {
		return "", err
	}

	return texts.EventDeleted, h.reply.EditMenu(ctx, query, texts.EventDeleted, keyboard.Back(eventsPage))
}

func (h *Handler) kick(ctx context.Context, query telego.CallbackQuery, a callback.Action) (string, error) {
	if !h.confirms.Confirm(callback.ActionKick, a.EventID, query.From.ID, a.UserID) {
		return texts.KickPressAgain, nil
	}

	removed, err := h.queues.RemoveParticipant(ctx, a.EventID, query.From.ID, a.UserID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back(a.EventsPage))
	}

	if err != nil {
		return "", err
	}

	text := texts.ParticipantRemoved

	if !removed {
		text = texts.ParticipantNotFound
	}

	return text, h.showMembers(ctx, query, a.EventID, a.Page, a.EventsPage)
}

func (h *Handler) showMembers(ctx context.Context, query telego.CallbackQuery, eventID int64, page, eventsPage int) error {
	view, err := h.queues.EventView(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return h.reply.EditMenu(ctx, query, texts.EventGone, keyboard.Back(eventsPage))
	}

	if err != nil {
		return err
	}

	kb, p := keyboard.Members(view, page, query.From.ID, eventsPage)

	return h.reply.EditMenu(ctx, query, texts.EventMembers(view, p.Index, p.Count), kb)
}
