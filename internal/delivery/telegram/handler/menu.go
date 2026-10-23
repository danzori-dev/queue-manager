package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func (h *Handler) HandleEvents(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.allowed(ctx, msg, access.RoleAdmin); !ok {
		return err
	}

	events, err := h.queue.Events(ctx)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return h.sendHTML(ctx, msg, texts.EventsEmpty)
	}

	return h.sendMenu(ctx, msg, texts.EventsTitle, eventsKeyboard(events))
}

func (h *Handler) HandleMenuCallback(ctx *th.Context, query telego.CallbackQuery) error {
	if !middleware.RoleFromContext(ctx).AtLeast(access.RoleAdmin) {
		return h.answer(ctx, query, texts.AccessDenied)
	}

	action, err := domain.ParseAction(query.Data)
	if err != nil {
		return h.answer(ctx, query, "")
	}

	text, err := h.applyMenuAction(ctx, query, action)
	if err != nil {
		_ = h.answer(ctx, query, texts.InternalError)

		return err
	}

	return h.answer(ctx, query, text)
}

func (h *Handler) applyMenuAction(ctx context.Context, query telego.CallbackQuery, a domain.Action) (string, error) {
	switch a.Kind {
	case domain.ActionList:
		return "", h.showEvents(ctx, query)

	case domain.ActionMenu:
		return "", h.showMenu(ctx, query, a.EventID)

	case domain.ActionClose:
		return h.closeEvent(ctx, query, a.EventID)

	case domain.ActionOpen:
		return h.openEvent(ctx, query, a.EventID)

	case domain.ActionDelete:
		return "", h.showDeleteConfirm(ctx, query, a.EventID)

	case domain.ActionDeleteConfirm:
		return h.deleteEvent(ctx, query, a.EventID)

	case domain.ActionKick:
		return h.kick(ctx, query, a)

	case domain.ActionMembers:
		return "", h.showMembers(ctx, query, a.EventID, a.Page)

	case domain.ActionCarry:
		return h.confirmCarry(ctx, query, a.EventID)

	case domain.ActionCarryCancel:
		return h.cancelCarry(ctx, query, a.EventID)

	default:
		return "", domain.ErrUnknownAction
	}
}

func (h *Handler) showEvents(ctx context.Context, query telego.CallbackQuery) error {
	events, err := h.queue.Events(ctx)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return h.editMenu(ctx, query, texts.EventsEmpty, nil)
	}

	return h.editMenu(ctx, query, texts.EventsTitle, eventsKeyboard(events))
}

func (h *Handler) showMenu(ctx context.Context, query telego.CallbackQuery, eventID int64) error {
	view, err := h.queue.EventView(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return h.editMenu(ctx, query, texts.EventGone, backKeyboard())
	}

	if err != nil {
		return err
	}

	return h.editMenu(ctx, query, texts.EventMenu(view), menuKeyboard(view))
}

func (h *Handler) closeEvent(ctx context.Context, query telego.CallbackQuery, eventID int64) (string, error) {
	err := h.queue.CloseEvent(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.editMenu(ctx, query, texts.EventGone, backKeyboard())
	}

	if err != nil {
		return "", err
	}

	return texts.EventClosedAnswer, h.showMenu(ctx, query, eventID)
}

func (h *Handler) openEvent(ctx context.Context, query telego.CallbackQuery, eventID int64) (string, error) {
	err := h.queue.OpenEvent(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.editMenu(ctx, query, texts.EventGone, backKeyboard())
	}

	if err != nil {
		return "", err
	}

	return texts.EventOpenedAnswer, h.showMenu(ctx, query, eventID)
}

func (h *Handler) showDeleteConfirm(ctx context.Context, query telego.CallbackQuery, eventID int64) error {
	view, err := h.queue.EventView(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return h.editMenu(ctx, query, texts.EventGone, backKeyboard())
	}

	if err != nil {
		return err
	}

	return h.editMenu(ctx, query, texts.DeleteConfirm(view.Event.Title), deleteConfirmKeyboard(eventID))
}

func (h *Handler) deleteEvent(ctx context.Context, query telego.CallbackQuery, eventID int64) (string, error) {
	err := h.queue.DeleteEvent(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.editMenu(ctx, query, texts.EventGone, backKeyboard())
	}

	if err != nil {
		return "", err
	}

	return texts.EventDeleted, h.editMenu(ctx, query, texts.EventDeleted, backKeyboard())
}

func (h *Handler) kick(ctx context.Context, query telego.CallbackQuery, a domain.Action) (string, error) {
	if !h.confirms.Confirm(domain.ActionKick, a.EventID, query.From.ID, a.UserID) {
		return texts.KickPressAgain, nil
	}

	removed, err := h.queue.RemoveParticipant(ctx, a.EventID, query.From.ID, a.UserID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.editMenu(ctx, query, texts.EventGone, backKeyboard())
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
	view, err := h.queue.EventView(ctx, eventID)
	if errors.Is(err, domain.ErrEventNotFound) {
		return h.editMenu(ctx, query, texts.EventGone, backKeyboard())
	}

	if err != nil {
		return err
	}

	p := paginate(len(view.Lines), page)

	return h.editMenu(ctx, query, texts.EventMembers(view, p.index, p.count), membersKeyboard(view, p, query.From.ID))
}

func eventsKeyboard(events []domain.Event) *telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0, len(events))

	for _, e := range events {
		label := fmt.Sprintf("#%d %s", e.ID, e.Title)
		if e.Status == domain.StatusClosed {
			label = "🔒 " + label
		}

		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					label, domain.Action{
						Kind:    domain.ActionMenu,
						EventID: e.ID,
					},
				),
			),
		)
	}

	return tu.InlineKeyboard(rows...)
}

func menuKeyboard(view appqueue.View) *telego.InlineKeyboardMarkup {
	eventID := view.Event.ID

	toggle := button(
		texts.ButtonClose, domain.Action{
			Kind:    domain.ActionClose,
			EventID: eventID,
		},
	)

	if view.Event.Status == domain.StatusClosed {
		toggle = button(
			texts.ButtonOpen, domain.Action{
				Kind:    domain.ActionOpen,
				EventID: eventID,
			},
		)
	}

	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			toggle, button(
				texts.ButtonRefresh, domain.Action{
					Kind:    domain.ActionMenu,
					EventID: eventID,
				},
			),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonMembers, domain.Action{
					Kind:    domain.ActionMembers,
					EventID: eventID,
				},
			),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonDelete, domain.Action{
					Kind:    domain.ActionDelete,
					EventID: eventID,
				},
			),
		),
		tu.InlineKeyboardRow(button(texts.ButtonBack, domain.Action{Kind: domain.ActionList})),
	)
}

func deleteConfirmKeyboard(eventID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonConfirmDelete, domain.Action{
					Kind:    domain.ActionDeleteConfirm,
					EventID: eventID,
				},
			),
			button(
				texts.ButtonCancel, domain.Action{
					Kind:    domain.ActionMenu,
					EventID: eventID,
				},
			),
		),
	)
}

func membersKeyboard(view appqueue.View, p page, selfID int64) *telego.InlineKeyboardMarkup {
	eventID := view.Event.ID
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+2)

	for _, line := range view.Lines[p.start:p.end] {
		label := fmt.Sprintf("❌ %d. %s", line.Position, line.Name)
		if line.UserID == selfID {
			label += " (ты)"
		}

		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					label, domain.Action{
						Kind:    domain.ActionKick,
						EventID: eventID,
						UserID:  line.UserID,
						Page:    p.index,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) domain.Action {
			return domain.Action{
				Kind:    domain.ActionMembers,
				EventID: eventID,
				Page:    index,
			}
		},
	)

	rows = append(
		rows, tu.InlineKeyboardRow(
			button(
				texts.ButtonBackToEvent, domain.Action{
					Kind:    domain.ActionMenu,
					EventID: eventID,
				},
			),
		),
	)

	return tu.InlineKeyboard(rows...)
}

func backKeyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(tu.InlineKeyboardRow(button(texts.ButtonBack, domain.Action{Kind: domain.ActionList})))
}

func button(label string, action domain.Action) telego.InlineKeyboardButton {
	return tu.InlineKeyboardButton(label).WithCallbackData(action.String())
}
