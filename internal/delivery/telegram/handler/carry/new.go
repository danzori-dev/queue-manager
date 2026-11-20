package carry

import (
	"context"
	"errors"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/keyboard"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/server/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func (h *Handler) requestNewTitle(ctx context.Context, query telego.CallbackQuery, draft *carryDraft, fromID int64, fromPos int) error {
	if draft.fromID != fromID || fromPos < 1 {
		return nil
	}

	kb := keyboard.CarryBackToPlaces(fromID, keyboard.PageOf(fromPos-1))
	if err := h.reply.EditMenu(ctx, query, texts.CarryNewTitlePrompt, kb); err != nil {
		return err
	}

	draft.fromPos = fromPos
	draft.awaitingTitle = true

	return nil
}

func (h *Handler) NewTitle(ctx *th.Context, msg telego.Message) (bool, error) {
	if msg.From == nil || !middleware.RoleFromContext(ctx).AtLeast(access.RoleAdmin) {
		return false, nil
	}

	adminID := msg.From.ID
	draft, err := h.carries.BeginTitle(adminID, msg.Chat.ID)

	switch {
	case errors.Is(err, errNoCarryTitle):
		return false, nil
	case errors.Is(err, errCarryExpired):
		return true, h.reply.SendHTML(ctx, msg, texts.CarryNewExpired)
	case errors.Is(err, errCarryProcessing):
		return true, h.reply.SendHTML(ctx, msg, texts.CarryProcessing)
	case err != nil:
		return true, err
	}

	defer func() {
		h.carries.Save(adminID, draft)
	}()

	plan, err := h.queues.PlanCarryNew(ctx, draft.fromID, draft.fromPos, msg.Text)
	switch {
	case errors.Is(err, domain.ErrEmptyTitle):
		return true, h.reply.SendHTML(ctx, msg, texts.TitleEmpty)
	case errors.Is(err, domain.ErrTitleTooLong):
		return true, h.reply.SendHTML(ctx, msg, texts.EventTitleTooLong)
	case err != nil:
		text, known := errorText(err)
		if !known {
			return true, err
		}

		kb := keyboard.CarryBackToSources(draft.fromID, draft.sourcesPage)

		if errors.Is(err, domain.ErrCarryPosition) {
			kb = keyboard.CarryBackToPlaces(draft.fromID, keyboard.PageOf(draft.fromPos-1))
		}

		if err := h.reply.EditMenuMessage(ctx, draft.chatID, draft.messageID, text, kb); err != nil {
			return true, err
		}

		draft.resetPlan()

		return true, nil
	}

	next := draft
	next.awaitingTitle = false
	next.setPlan(plan)

	text, kb := planMenu(&next, 0)
	if err := h.reply.EditMenuMessage(ctx, next.chatID, next.messageID, text, kb); err != nil {
		return true, err
	}

	draft = next

	h.reply.DeleteMessage(ctx, msg.Chat.ID, msg.MessageID)

	return true, nil
}

func (h *Handler) confirmNew(
	ctx context.Context, query telego.CallbackQuery, draft *carryDraft,
) (string, error) {
	adminID := query.From.ID
	err := h.queues.CreateCarriedEvent(ctx, *draft.plan, adminID)

	switch {
	case err == nil:
		defer h.carries.Delete(adminID)

		return texts.EventCreated, h.reply.EditMenu(ctx, query, texts.EventCreated, keyboard.None())

	case errors.Is(err, domain.ErrTargetNotSet):
		h.carries.Save(adminID, *draft)

		return texts.QueueTopicNotSet, nil

	case errors.Is(err, domain.ErrCarryChanged):
		return h.refreshNewPlan(ctx, query, draft)

	case errors.Is(err, domain.ErrCarryPosition):
		draft.resetPlan()
		page := keyboard.PageOf(draft.fromPos - 1)

		err := h.showPlaces(ctx, query, draft, draft.fromID, page)

		h.carries.Save(adminID, *draft)

		return texts.CarryPosition, err

	case errors.Is(err, domain.ErrEventNotFound), errors.Is(err, domain.ErrCarrySourceOpen):
		err = h.stepFailed(ctx, query, draft, err)

		h.carries.Save(adminID, *draft)

		return "", err

	default:
		h.carries.Delete(adminID)

		return "", err
	}
}

func (h *Handler) refreshNewPlan(
	ctx context.Context, query telego.CallbackQuery, draft *carryDraft,
) (string, error) {
	defer func() {
		h.carries.Save(query.From.ID, *draft)
	}()

	plan, err := h.queues.PlanCarryNew(ctx, draft.fromID, draft.fromPos, draft.plan.To.Title)
	if errors.Is(err, domain.ErrCarryPosition) {
		draft.resetPlan()
		page := keyboard.PageOf(draft.fromPos - 1)

		return texts.CarryPosition, h.showPlaces(ctx, query, draft, draft.fromID, page)
	}

	if err != nil {
		return "", h.stepFailed(ctx, query, draft, err)
	}

	draft.setPlan(plan)

	return texts.CarryChanged, h.renderPlan(ctx, query, draft, 0)
}
