package carry

import (
	"context"
	"errors"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/keyboard"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/reply"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/server/middleware"
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

	draft, err := h.carries.Start(msg.From.ID, msg.Chat.ID)
	if err != nil {
		return h.reply.SendHTML(ctx, msg, texts.CarryProcessing)
	}

	events, err := h.queues.CarrySources(ctx)
	if err != nil {
		h.carries.Delete(msg.From.ID)

		return err
	}

	if len(events) == 0 {
		err := h.reply.SendHTML(ctx, msg, texts.CarrySourcesEmpty)
		h.carries.Delete(msg.From.ID)

		return err
	}

	kb, p := keyboard.CarrySources(events, 0, 0)

	messageID, err := h.reply.SendMenuWithID(ctx, msg, texts.CarryChooseSource(p.Index, p.Count), kb)
	if err != nil {
		h.carries.Delete(msg.From.ID)

		return err
	}

	draft.messageID = messageID
	h.carries.Save(msg.From.ID, draft)

	return nil
}

func (h *Handler) Callback(ctx *th.Context, query telego.CallbackQuery) error {
	if !middleware.RoleFromContext(ctx).AtLeast(access.RoleAdmin) {
		return h.reply.Answer(ctx, query, texts.AccessDenied)
	}

	action, err := callback.ParseAction(query.Data)
	if err != nil {
		return h.reply.Answer(ctx, query, "")
	}

	switch action.Kind {
	case callback.ActionCarry, callback.ActionCarryPlanPage:
		if action.ConfirmationID == 0 {
			return h.reply.Answer(ctx, query, texts.CarryExpired)
		}
	}

	draft, err := h.carries.Begin(query.From.ID, query.Message.GetChat().ID, query.Message.GetMessageID(), action.ConfirmationID)
	if errors.Is(err, errCarryProcessing) {
		return h.reply.Answer(ctx, query, texts.CarryProcessing)
	}

	if err != nil {
		return h.reply.Answer(ctx, query, texts.CarryExpired)
	}

	text, err := h.applyAction(ctx, query, &draft, action)
	if err != nil {
		_ = h.reply.Answer(ctx, query, texts.InternalError)

		return err
	}

	return h.reply.Answer(ctx, query, text)
}

func (h *Handler) applyAction(ctx context.Context, query telego.CallbackQuery, draft *carryDraft, a callback.Action) (string, error) {
	switch a.Kind {
	case callback.ActionCarry:
		return h.confirm(ctx, query, draft, a.EventID)

	case callback.ActionCarryCancel:
		return h.cancel(ctx, query, draft, a.EventID)
	}

	defer func() {
		h.carries.Save(query.From.ID, *draft)
	}()

	switch a.Kind {
	case callback.ActionCarrySourcesPage:
		draft.resetPlan()

		return "", h.showSources(ctx, query, draft, a.Page)

	case callback.ActionCarrySource:
		draft.resetPlan()

		return "", h.showPlaces(ctx, query, draft, a.EventID, 0)

	case callback.ActionCarryPlace:
		draft.resetPlan()

		return "", h.showTargets(ctx, query, draft, a.EventID, a.Position, 0)

	case callback.ActionCarryTargetsPage:
		draft.resetPlan()

		return "", h.showTargets(ctx, query, draft, a.EventID, a.Position, a.Page)

	case callback.ActionCarryTarget:
		draft.resetPlan()

		return "", h.showPlan(ctx, query, draft, a.EventID)

	case callback.ActionCarryPlacesPage:
		draft.resetPlan()

		return "", h.showPlaces(ctx, query, draft, a.EventID, a.Page)

	case callback.ActionCarryPlanPage:
		return "", h.showPlanPage(ctx, query, draft, a.EventID, a.Page)

	default:
		return "", callback.ErrUnknownAction
	}
}

func (h *Handler) showSources(ctx context.Context, query telego.CallbackQuery, draft *carryDraft, page int) error {
	events, err := h.queues.CarrySources(ctx)
	if err != nil {
		return h.stepFailed(ctx, query, draft, err)
	}

	kb, p := keyboard.CarrySources(events, page, draft.fromID)
	draft.sourcesPage = p.Index

	if len(events) == 0 {
		return h.reply.EditMenu(ctx, query, texts.CarrySourcesEmpty, kb)
	}

	return h.reply.EditMenu(ctx, query, texts.CarryChooseSource(p.Index, p.Count), kb)
}

func (h *Handler) showPlaces(ctx context.Context, query telego.CallbackQuery, draft *carryDraft, fromID int64, page int) error {
	view, err := h.queues.CarrySource(ctx, fromID)
	if err != nil {
		return h.stepFailed(ctx, query, draft, err)
	}

	if draft.fromID != fromID {
		draft.fromID = fromID
		draft.fromPos = 0
		draft.targetsPage = 0
	}

	if len(view.Lines) == 0 {
		return h.reply.EditMenu(ctx, query, texts.CarrySourceEmpty, keyboard.CarryBackToSources(fromID, draft.sourcesPage))
	}

	kb, p := keyboard.CarryPlaces(view, page, draft.sourcesPage)

	return h.reply.EditMenu(ctx, query, texts.CarryChoosePlace(view.Event.Title, p.Index, p.Count), kb)
}

func (h *Handler) showTargets(ctx context.Context, query telego.CallbackQuery, draft *carryDraft, fromID int64, fromPos, page int) error {
	if draft.fromID != fromID {
		return nil
	}

	events, err := h.queues.CarryTargets(ctx, fromID, fromPos)
	if err != nil {
		return h.stepFailed(ctx, query, draft, err)
	}

	kb, p := keyboard.CarryTargets(events, fromID, fromPos, page)

	draft.fromPos = fromPos
	draft.targetsPage = p.Index

	if len(events) == 0 {
		return h.reply.EditMenu(ctx, query, texts.CarryTargetsEmpty, kb)
	}

	return h.reply.EditMenu(ctx, query, texts.CarryChooseTarget(fromID, fromPos, p.Index, p.Count), kb)
}

func (h *Handler) showPlan(ctx context.Context, query telego.CallbackQuery, draft *carryDraft, toID int64) error {
	if draft.fromID == 0 || draft.fromPos < 1 {
		return nil
	}

	plan, err := h.queues.PlanCarry(ctx, draft.fromID, draft.fromPos, toID)
	if err != nil {
		return h.stepFailed(ctx, query, draft, err)
	}

	if len(plan.Lines) == 0 {
		return h.stepFailed(ctx, query, draft, domain.ErrCarryPosition)
	}

	draft.setPlan(plan)

	return h.renderPlan(ctx, query, draft, 0)
}

func (h *Handler) showPlanPage(ctx context.Context, query telego.CallbackQuery, draft *carryDraft, toID int64, page int) error {
	if draft.plan == nil || draft.toID != toID {
		return nil
	}

	return h.renderPlan(ctx, query, draft, page)
}

func (h *Handler) confirm(ctx context.Context, query telego.CallbackQuery, draft *carryDraft, fromID int64) (string, error) {
	if draft.plan == nil || draft.fromID != fromID {
		h.carries.Save(query.From.ID, *draft)

		return texts.CarryExpired, nil
	}

	defer h.carries.Delete(query.From.ID)

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

func (h *Handler) cancel(ctx context.Context, query telego.CallbackQuery, draft *carryDraft, fromID int64) (string, error) {
	if draft.fromID != fromID {
		h.carries.Save(query.From.ID, *draft)

		return texts.CarryExpired, nil
	}

	defer h.carries.Delete(query.From.ID)

	return texts.CarryCanceled, h.reply.EditMenu(ctx, query, texts.CarryCanceled, keyboard.None())
}

func (h *Handler) renderPlan(ctx context.Context, query telego.CallbackQuery, draft *carryDraft, page int) error {
	plan := *draft.plan

	kb, p := keyboard.Carry(plan, draft.targetsPage, page, draft.confirmationID)
	start, end := p.Bounds()

	firstPosition := plan.Lines[0].Position
	lines := make([]appqueue.Line, 0)

	for _, line := range plan.Lines {
		index := line.Position - firstPosition

		if index >= start && index < end {
			lines = append(lines, line)
		}
	}

	return h.reply.EditMenu(ctx, query, texts.CarryPlan(plan, lines, p.Index, p.Count), kb)
}

func (h *Handler) stepFailed(ctx context.Context, query telego.CallbackQuery, draft *carryDraft, err error) error {
	text, known := errorText(err)
	if !known {
		return err
	}

	draft.resetPlan()

	return h.reply.EditMenu(ctx, query, text, keyboard.CarryBackToSources(draft.fromID, draft.sourcesPage))
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
