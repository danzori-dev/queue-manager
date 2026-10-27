package carry

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/command"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/keyboard"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/reply"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type Handler struct {
	reply   *reply.Replier
	queues  QueueService
	carries *carryDrafts
}

func New(replier *reply.Replier, queues QueueService) *Handler {
	return &Handler{
		reply:   replier,
		queues:  queues,
		carries: newCarryDrafts(),
	}
}

func (h *Handler) Carry(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleAdmin); !ok {
		return err
	}

	draft, ok := parseArgs(command.Args(msg.Text))
	if !ok {
		return h.reply.SendHTML(ctx, msg, texts.CarryUsage)
	}

	plan, err := h.queues.PlanCarry(ctx, draft.fromID, draft.fromPos, draft.toID)
	if err != nil {
		text, known := errorText(err)
		if !known {
			return err
		}

		return h.reply.SendHTML(ctx, msg, text)
	}

	h.carries.Put(msg.From.ID, draft)

	return h.reply.SendMenu(ctx, msg, texts.CarryPlan(plan), keyboard.Carry(draft.fromID))
}

func (h *Handler) Confirm(ctx context.Context, query telego.CallbackQuery, fromID int64) (string, error) {
	draft, ok := h.carries.Take(query.From.ID, fromID)
	if !ok {
		return texts.CarryExpired, h.reply.EditMenu(ctx, query, texts.CarryExpired, keyboard.None())
	}

	moved, err := h.queues.Carry(ctx, draft.fromID, draft.fromPos, draft.toID, query.From.ID)
	if err != nil {
		text, known := errorText(err)
		if !known {
			return "", err
		}

		return text, h.reply.EditMenu(ctx, query, text, keyboard.None())
	}

	return texts.CarryDoneAnswer, h.reply.EditMenu(ctx, query, texts.Carried(moved), keyboard.None())
}

func (h *Handler) Cancel(ctx context.Context, query telego.CallbackQuery, fromID int64) (string, error) {
	h.carries.Take(query.From.ID, fromID)

	return texts.CarryCanceled, h.reply.EditMenu(ctx, query, texts.CarryCanceled, keyboard.None())
}

func parseArgs(args string) (carryDraft, bool) {
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

func errorText(err error) (string, bool) {
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
