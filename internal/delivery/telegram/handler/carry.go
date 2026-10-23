package handler

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func (h *Handler) HandleCarry(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.allowed(ctx, msg, access.RoleAdmin); !ok {
		return err
	}

	draft, ok := parseCarryArgs(commandArgs(msg.Text))
	if !ok {
		return h.sendHTML(ctx, msg, texts.CarryUsage)
	}

	plan, err := h.queue.PlanCarry(ctx, draft.fromID, draft.fromPos, draft.toID)
	if err != nil {
		text, known := carryErrorText(err)
		if !known {
			return err
		}

		return h.sendHTML(ctx, msg, text)
	}

	h.carries.Put(msg.From.ID, draft)

	return h.sendMenu(ctx, msg, texts.CarryPlan(plan), carryKeyboard(draft.fromID))
}

func (h *Handler) confirmCarry(ctx context.Context, query telego.CallbackQuery, fromID int64) (string, error) {
	draft, ok := h.carries.Take(query.From.ID, fromID)
	if !ok {
		return texts.CarryExpired, h.editMenu(ctx, query, texts.CarryExpired, noButtons())
	}

	moved, err := h.queue.Carry(ctx, draft.fromID, draft.fromPos, draft.toID)
	if err != nil {
		text, known := carryErrorText(err)
		if !known {
			return "", err
		}

		return text, h.editMenu(ctx, query, text, noButtons())
	}

	return texts.CarryDoneAnswer, h.editMenu(ctx, query, texts.Carried(moved), noButtons())
}

func (h *Handler) cancelCarry(ctx context.Context, query telego.CallbackQuery, fromID int64) (string, error) {
	h.carries.Take(query.From.ID, fromID)

	return texts.CarryCanceled, h.editMenu(ctx, query, texts.CarryCanceled, noButtons())
}

func parseCarryArgs(args string) (carryDraft, bool) {
	fields := strings.Fields(args)
	if len(fields) != 3 {
		return carryDraft{}, false
	}

	fromID, fromErr := strconv.ParseInt(fields[0], 10, 64)
	fromPos, posErr := strconv.Atoi(fields[1])
	toID, toErr := strconv.ParseInt(fields[2], 10, 64)

	if errors.Join(fromErr, posErr, toErr) != nil {
		return carryDraft{}, false
	}

	return carryDraft{
		fromID:  fromID,
		fromPos: fromPos,
		toID:    toID,
	}, true
}

func carryErrorText(err error) (string, bool) {
	switch {
	case errors.Is(err, domain.ErrEventNotFound):
		return texts.QueueNotFound, true

	case errors.Is(err, domain.ErrCarrySameEvent):
		return texts.CarrySameEvent, true

	case errors.Is(err, domain.ErrCarrySourceOpen):
		return texts.CarrySourceOpen, true

	case errors.Is(err, domain.ErrCarryTargetClosed):
		return texts.CarryTargetClosed, true

	case errors.Is(err, domain.ErrCarryPosition):
		return texts.CarryPosition, true

	default:
		return "", false
	}
}

func carryKeyboard(fromID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonCarry, domain.Action{
					Kind:    domain.ActionCarry,
					EventID: fromID,
				},
			),
			button(
				texts.ButtonCancel, domain.Action{
					Kind:    domain.ActionCarryCancel,
					EventID: fromID,
				},
			),
		),
	)
}
