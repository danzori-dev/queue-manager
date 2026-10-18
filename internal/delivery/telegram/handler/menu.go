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
	var (
		text string
		err  error
	)

	switch a.Kind {
	case domain.ActionList:
		return "", h.showEvents(ctx, query)

	case domain.ActionMenu:
		return "", h.showMenu(ctx, query, a.EventID)

	case domain.ActionDelete:
		return "", h.showDeleteConfirm(ctx, query, a.EventID)

	case domain.ActionClose:
		text, err = texts.EventClosedAnswer, h.queue.CloseEvent(ctx, a.EventID)

	case domain.ActionOpen:
		text, err = texts.EventOpenedAnswer, h.queue.OpenEvent(ctx, a.EventID)

	case domain.ActionKick:
		var removed bool

		removed, err = h.queue.RemoveParticipant(ctx, a.EventID, query.From.ID, a.UserID)

		text = texts.ParticipantRemoved

		if !removed {
			text = texts.ParticipantNotFound
		}

	case domain.ActionDeleteConfirm:
		err = h.queue.DeleteEvent(ctx, a.EventID)
		if err == nil {
			return texts.EventDeletedAnswer, h.editMenu(ctx, query, texts.EventDeleted, backKeyboard())
		}

	default:
		return "", domain.ErrUnknownAction
	}

	if errors.Is(err, domain.ErrEventNotFound) {
		return texts.QueueNotFound, h.editMenu(ctx, query, texts.EventGone, backKeyboard())
	}

	if err != nil {
		return "", err
	}

	return text, h.showMenu(ctx, query, a.EventID)
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

func eventsKeyboard(events []domain.Event) *telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0, len(events))

	for _, e := range events {
		label := e.Title
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
	id := view.Event.ID
	rows := make([][]telego.InlineKeyboardButton, 0, len(view.Lines)+3)

	for _, line := range view.Lines {
		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					fmt.Sprintf("❌ %d. %s", line.Position, line.Name), domain.Action{
						Kind:    domain.ActionKick,
						EventID: id,
						UserID:  line.UserID,
					},
				),
			),
		)
	}

	toggle := button(
		texts.ButtonClose, domain.Action{
			Kind:    domain.ActionClose,
			EventID: id,
		},
	)

	if view.Event.Status == domain.StatusClosed {
		toggle = button(
			texts.ButtonOpen, domain.Action{
				Kind:    domain.ActionOpen,
				EventID: id,
			},
		)
	}

	rows = append(
		rows,
		tu.InlineKeyboardRow(
			toggle, button(
				texts.ButtonRefresh, domain.Action{
					Kind:    domain.ActionMenu,
					EventID: id,
				},
			),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonDelete, domain.Action{
					Kind:    domain.ActionDelete,
					EventID: id,
				},
			),
		),
		tu.InlineKeyboardRow(button(texts.ButtonBack, domain.Action{Kind: domain.ActionList})),
	)

	return tu.InlineKeyboard(rows...)
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

func backKeyboard() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(tu.InlineKeyboardRow(button(texts.ButtonBack, domain.Action{Kind: domain.ActionList})))
}

func button(label string, action domain.Action) telego.InlineKeyboardButton {
	return tu.InlineKeyboardButton(label).WithCallbackData(action.String())
}
