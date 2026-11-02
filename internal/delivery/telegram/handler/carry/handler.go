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

	h.carries.Put(msg.From.ID, carryDraft{})

	events, err := h.queues.CarrySources(ctx)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return h.reply.SendHTML(ctx, msg, texts.CarrySourcesEmpty)
	}

	kb, p := keyboard.CarrySources(events, 0, 0)

	return h.reply.SendMenu(ctx, msg, texts.CarryChooseSource(p.Index, p.Count), kb)
}

func (h *Handler) Callback(ctx *th.Context, query telego.CallbackQuery) error {
	if !middleware.RoleFromContext(ctx).AtLeast(access.RoleAdmin) {
		return h.reply.Answer(ctx, query, texts.AccessDenied)
	}

	action, err := callback.ParseAction(query.Data)
	if err != nil {
		return h.reply.Answer(ctx, query, "")
	}

	text, err := h.applyAction(ctx, query, action)
	if err != nil {
		_ = h.reply.Answer(ctx, query, texts.InternalError)

		return err
	}

	return h.reply.Answer(ctx, query, text)
}

func (h *Handler) applyAction(ctx context.Context, query telego.CallbackQuery, a callback.Action) (string, error) {
	switch a.Kind {
	case callback.ActionCarrySourcesPage:
		return "", h.showSources(ctx, query, a.Page)

	case callback.ActionCarrySource:
		return "", h.showPlaces(ctx, query, a.EventID, 0)

	case callback.ActionCarryPlace:
		return "", h.showTargets(ctx, query, a.EventID, a.Position, 0)

	case callback.ActionCarryTargetsPage:
		return "", h.showTargets(ctx, query, a.EventID, a.Position, a.Page)

	case callback.ActionCarryTarget:
		return "", h.showPlan(ctx, query, a.EventID)

	case callback.ActionCarryPlacesPage:
		return "", h.showPlaces(ctx, query, a.EventID, a.Page)

	case callback.ActionCarryPlanPage:
		return "", h.showPlanPage(ctx, query, a.EventID, a.Page)

	case callback.ActionCarry:
		return h.confirm(ctx, query, a.EventID)

	case callback.ActionCarryCancel:
		return h.cancel(ctx, query, a.EventID)

	default:
		return "", callback.ErrUnknownAction
	}
}

func (h *Handler) showSources(ctx context.Context, query telego.CallbackQuery, page int) error {
	draft, ok := h.carries.Get(query.From.ID)
	if !ok {
		return h.reply.EditMenu(ctx, query, texts.CarryExpired, keyboard.None())
	}

	events, err := h.queues.CarrySources(ctx)
	if err != nil {
		return h.stepFailed(ctx, query, err)
	}

	if len(events) == 0 {
		return h.reply.EditMenu(ctx, query, texts.CarrySourcesEmpty, keyboard.CarryCancel(draft.fromID))
	}

	kb, p := keyboard.CarrySources(events, page, draft.fromID)

	return h.reply.EditMenu(ctx, query, texts.CarryChooseSource(p.Index, p.Count), kb)
}

func (h *Handler) showPlaces(ctx context.Context, query telego.CallbackQuery, fromID int64, page int) error {
	view, err := h.queues.CarrySource(ctx, fromID)
	if err != nil {
		return h.stepFailed(ctx, query, err)
	}

	if !h.carries.SelectSource(query.From.ID, fromID) {
		return h.reply.EditMenu(ctx, query, texts.CarryExpired, keyboard.None())
	}

	if len(view.Lines) == 0 {
		return h.reply.EditMenu(ctx, query, texts.CarrySourceEmpty, keyboard.CarryBackToSources(fromID))
	}

	kb, p := keyboard.CarryPlaces(view, page)

	return h.reply.EditMenu(ctx, query, texts.CarryChoosePlace(view.Event.Title, p.Index, p.Count), kb)
}

func (h *Handler) showTargets(ctx context.Context, query telego.CallbackQuery, fromID int64, fromPos, page int) error {
	events, err := h.queues.CarryTargets(ctx, fromID, fromPos)
	if err != nil {
		return h.stepFailed(ctx, query, err)
	}

	kb, p := keyboard.CarryTargets(events, fromID, fromPos, page)

	if !h.carries.SelectPlace(query.From.ID, fromID, fromPos, p.Index) {
		return h.reply.EditMenu(ctx, query, texts.CarryExpired, keyboard.None())
	}

	if len(events) == 0 {
		return h.reply.EditMenu(ctx, query, texts.CarryTargetsEmpty, kb)
	}

	return h.reply.EditMenu(ctx, query, texts.CarryChooseTarget(fromID, fromPos, p.Index, p.Count), kb)
}

func (h *Handler) showPlan(ctx context.Context, query telego.CallbackQuery, toID int64) error {
	draft, ok := h.carries.Get(query.From.ID)
	if !ok {
		return h.reply.EditMenu(ctx, query, texts.CarryExpired, keyboard.None())
	}

	if draft.fromID == 0 || draft.fromPos < 1 {
		return nil
	}

	plan, err := h.queues.PlanCarry(ctx, draft.fromID, draft.fromPos, toID)
	if err != nil {
		return h.stepFailed(ctx, query, err)
	}

	draft, ok = h.carries.SetPlan(query.From.ID, draft.fromPos, plan)
	if !ok {
		return nil
	}

	return h.renderPlan(ctx, query, draft, 0)
}

func (h *Handler) showPlanPage(ctx context.Context, query telego.CallbackQuery, toID int64, page int) error {
	draft, ok := h.carries.Get(query.From.ID)
	if !ok {
		return h.reply.EditMenu(ctx, query, texts.CarryExpired, keyboard.None())
	}

	if draft.plan == nil || draft.toID != toID {
		return nil
	}

	return h.renderPlan(ctx, query, draft, page)
}

func (h *Handler) confirm(ctx context.Context, query telego.CallbackQuery, fromID int64) (string, error) {
	draft, ok := h.carries.TakePlan(query.From.ID, fromID)
	if !ok {
		return texts.CarryExpired, nil
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

func (h *Handler) cancel(ctx context.Context, query telego.CallbackQuery, fromID int64) (string, error) {
	if _, ok := h.carries.Take(query.From.ID, fromID); !ok {
		return texts.CarryExpired, nil
	}

	return texts.CarryCanceled, h.reply.EditMenu(ctx, query, texts.CarryCanceled, keyboard.None())
}

func (h *Handler) renderPlan(ctx context.Context, query telego.CallbackQuery, draft carryDraft, page int) error {
	plan := *draft.plan

	kb, p := keyboard.Carry(plan, draft.targetsPage, page)
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

func (h *Handler) stepFailed(ctx context.Context, query telego.CallbackQuery, err error) error {
	text, known := errorText(err)
	if !known {
		return err
	}

	kb := keyboard.None()

	if draft, ok := h.carries.Get(query.From.ID); ok {
		kb = keyboard.CarryBackToSources(draft.fromID)
	}

	return h.reply.EditMenu(ctx, query, text, kb)
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
