package swap

import (
	"context"
	"errors"
	"slices"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/keyboard"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/reply"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type Handler struct {
	reply *reply.Replier
	swaps SwapService
}

func New(replier *reply.Replier, swaps SwapService) *Handler {
	return &Handler{
		reply: replier,
		swaps: swaps,
	}
}

func (h *Handler) Swap(ctx *th.Context, msg telego.Message) error {
	events, err := h.swaps.SwapTargets(ctx, msg.From.ID)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return h.reply.SendHTML(ctx, msg, texts.SwapNotQueues)
	}

	return h.reply.SendMenu(ctx, msg, texts.SwapChooseQueue, keyboard.SwapEvents(events))
}

func (h *Handler) Callback(ctx *th.Context, query telego.CallbackQuery) error {
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

func (h *Handler) Start(ctx context.Context, msg telego.Message, eventID int64) error {
	view, err := h.swaps.SwapCandidates(ctx, eventID, msg.From.ID)
	if err != nil {
		text, ok := errorText(err)
		if !ok {
			return err
		}

		return h.reply.SendHTML(ctx, msg, text)
	}

	if !hasOtherPlaces(view, msg.From.ID) {
		return h.reply.SendHTML(ctx, msg, texts.SwapNobody)
	}

	kb, p := keyboard.SwapPeople(view, msg.From.ID, 0)

	return h.reply.SendMenu(ctx, msg, texts.SwapChoosePerson(view.Event.ID, view.Event.Title, p.Index, p.Count), kb)
}

func (h *Handler) applyAction(ctx context.Context, query telego.CallbackQuery, a callback.Action) (string, error) {
	switch a.Kind {
	case callback.ActionSwapEvent, callback.ActionSwapPage:
		return "", h.showCandidates(ctx, query, a.EventID, a.Page)

	case callback.ActionSwapPick:
		return "", h.showConfirm(ctx, query, a.EventID, a.UserID)

	case callback.ActionSwapSend:
		return "", h.send(ctx, query, a.EventID, a.UserID)

	case callback.ActionSwapAbort:
		return "", h.reply.EditMenu(ctx, query, texts.SwapAborted, keyboard.None())

	case callback.ActionSwapAccept, callback.ActionSwapDecline, callback.ActionSwapCancel:
		return h.resolve(ctx, query, a)

	default:
		return "", callback.ErrUnknownAction
	}
}

func (h *Handler) showCandidates(ctx context.Context, query telego.CallbackQuery, eventID int64, page int) error {
	view, err := h.swaps.SwapCandidates(ctx, eventID, query.From.ID)
	if err != nil {
		return h.stepFailed(ctx, query, err)
	}

	if !hasOtherPlaces(view, query.From.ID) {
		return h.reply.EditMenu(ctx, query, texts.SwapNobody, keyboard.None())
	}

	kb, p := keyboard.SwapPeople(view, query.From.ID, page)

	return h.reply.EditMenu(ctx, query, texts.SwapChoosePerson(view.Event.ID, view.Event.Title, p.Index, p.Count), kb)
}

func (h *Handler) showConfirm(ctx context.Context, query telego.CallbackQuery, eventID, toID int64) error {
	view, err := h.swaps.SwapCandidates(ctx, eventID, query.From.ID)
	if err != nil {
		return h.stepFailed(ctx, query, err)
	}

	me, okMe := findLine(view, query.From.ID)
	other, okOther := findLine(view, toID)

	if !okMe || !okOther {
		return h.reply.EditMenu(ctx, query, texts.ParticipantNotFound, keyboard.None())
	}

	return h.reply.EditMenu(
		ctx, query, texts.SwapConfirm(view.Event.Title, me.Position, other.Name, other.Position),
		keyboard.SwapConfirm(eventID, toID),
	)
}

func (h *Handler) send(ctx context.Context, query telego.CallbackQuery, eventID, toID int64) error {
	req, err := h.swaps.RequestSwap(ctx, eventID, query.From.ID, toID, query.Message.GetMessageID())
	if err != nil {
		return h.stepFailed(ctx, query, err)
	}

	return h.reply.EditMenu(ctx, query, texts.SwapSent(req.ToName), keyboard.SwapCancel(req.ID))
}

func (h *Handler) resolve(ctx context.Context, query telego.CallbackQuery, a callback.Action) (string, error) {
	switch a.Kind {
	case callback.ActionSwapAccept:
		return h.accept(ctx, query, a.EventID)

	case callback.ActionSwapDecline:
		req, err := h.swaps.DeclineSwap(ctx, a.EventID, query.From.ID)
		if err != nil {
			return h.gone(ctx, query, err)
		}

		h.reply.Notify(ctx, req.FromID, texts.SwapDeclinedForInitiator(req.ToName, req.Event.Title))
		h.reply.DeleteMessage(ctx, req.FromID, req.FromMessageID)

		return "", h.reply.EditMenu(ctx, query, texts.SwapDeclined, keyboard.None())

	default:
		req, err := h.swaps.CancelSwap(ctx, a.EventID, query.From.ID)
		if err != nil {
			return h.gone(ctx, query, err)
		}

		h.reply.DeleteMessage(ctx, req.ToID, req.ToMessageID)

		return "", h.reply.EditMenu(ctx, query, texts.SwapCanceled, keyboard.None())
	}
}

func (h *Handler) accept(ctx context.Context, query telego.CallbackQuery, requestID int64) (string, error) {
	req, err := h.swaps.AcceptSwap(ctx, requestID, query.From.ID)

	switch {
	case errors.Is(err, domain.ErrSwapChanged),
		errors.Is(err, domain.ErrEventClosed),
		errors.Is(err, domain.ErrEventNotFound):
		h.reply.Notify(ctx, req.FromID, texts.SwapFailedForInitiator(req.Event.Title))
		h.reply.DeleteMessage(ctx, req.FromID, req.FromMessageID)

		return "", h.reply.EditMenu(ctx, query, texts.SwapChanged, keyboard.None())

	case err != nil:
		return h.gone(ctx, query, err)
	}

	h.reply.Notify(ctx, req.FromID, texts.SwapAcceptedForInitiator(req.ToName, req.Event.Title, req.ToPos))
	h.reply.DeleteMessage(ctx, req.FromID, req.FromMessageID)

	return "", h.reply.EditMenu(ctx, query, texts.SwapAcceptedForAddressee(req.FromPos), keyboard.None())
}

func (h *Handler) gone(ctx context.Context, query telego.CallbackQuery, err error) (string, error) {
	switch {
	case errors.Is(err, domain.ErrSwapForbidden):
		return texts.SwapNotYours, nil

	case errors.Is(err, domain.ErrSwapNotFound), errors.Is(err, domain.ErrSwapExpired):
		return "", h.reply.EditMenu(ctx, query, texts.SwapGone, keyboard.None())

	default:
		return "", err
	}
}

func (h *Handler) stepFailed(ctx context.Context, query telego.CallbackQuery, err error) error {
	text, ok := errorText(err)
	if !ok {
		return err
	}

	return h.reply.EditMenu(ctx, query, text, keyboard.None())
}

func errorText(err error) (string, bool) {
	switch {
	case errors.Is(err, domain.ErrEventClosed):
		return texts.QueueClosed, true

	case errors.Is(err, domain.ErrEventNotFound):
		return texts.QueueNotFound, true

	case errors.Is(err, domain.ErrNotInQueue):
		return texts.SwapNotInQueue, true

	case errors.Is(err, domain.ErrSwapUnreachable):
		return texts.SwapUnreachable, true

	case errors.Is(err, domain.ErrSwapSelf):
		return texts.SwapSelf, true

	case errors.Is(err, domain.ErrSwapSamePlace):
		return texts.SwapSamePlace, true

	case errors.Is(err, domain.ErrSwapBusy):
		return texts.SwapBusy, true

	default:
		return "", false
	}
}

func hasOtherPlaces(view appqueue.View, userID int64) bool {
	me, ok := findLine(view, userID)

	return ok && slices.ContainsFunc(
		view.Lines, func(l appqueue.Line) bool {
			return l.Position != me.Position
		},
	)
}

func findLine(view appqueue.View, userID int64) (appqueue.Line, bool) {
	i := slices.IndexFunc(
		view.Lines, func(l appqueue.Line) bool {
			return l.UserID == userID
		},
	)
	if i < 0 {
		return appqueue.Line{}, false
	}

	return view.Lines[i], true
}
