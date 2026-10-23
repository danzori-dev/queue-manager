package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func (h *Handler) HandleSwap(ctx *th.Context, msg telego.Message) error {
	events, err := h.swaps.SwapTargets(ctx, msg.From.ID)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return h.sendHTML(ctx, msg, texts.SwapNotQueues)
	}

	return h.sendMenu(ctx, msg, texts.SwapChooseQueue, swapEventsKeyboard(events))
}

func (h *Handler) HandleSwapCallback(ctx *th.Context, query telego.CallbackQuery) error {
	action, err := domain.ParseAction(query.Data)
	if err != nil {
		return h.answer(ctx, query, "")
	}

	text, err := h.applySwapAction(ctx, query, action)
	if err != nil {
		_ = h.answer(ctx, query, texts.InternalError)

		return err
	}

	return h.answer(ctx, query, text)
}

func (h *Handler) applySwapAction(ctx context.Context, query telego.CallbackQuery, a domain.Action) (string, error) {
	switch a.Kind {
	case domain.ActionSwapEvent, domain.ActionSwapPage:
		return "", h.showSwapCandidates(ctx, query, a.EventID, a.Page)

	case domain.ActionSwapPick:
		return "", h.showSwapConfirm(ctx, query, a.EventID, a.UserID)

	case domain.ActionSwapSend:
		return "", h.sendSwap(ctx, query, a.EventID, a.UserID)

	case domain.ActionSwapAbort:
		return "", h.editMenu(ctx, query, texts.SwapAborted, noButtons())

	case domain.ActionSwapAccept, domain.ActionSwapDecline, domain.ActionSwapCancel:
		return h.resolveSwap(ctx, query, a)

	default:
		return "", domain.ErrUnknownAction
	}
}

func (h *Handler) showSwapCandidates(ctx context.Context, query telego.CallbackQuery, eventID int64, page int) error {
	view, err := h.swaps.SwapCandidates(ctx, eventID, query.From.ID)
	if err != nil {
		return h.swapStepFailed(ctx, query, err)
	}

	if len(view.Lines) < 2 {
		return h.editMenu(ctx, query, texts.SwapNobody, noButtons())
	}

	text, kb := swapPeople(view, query.From.ID, page)

	return h.editMenu(ctx, query, text, kb)
}

func (h *Handler) startSwap(ctx context.Context, msg telego.Message, eventID int64) error {
	view, err := h.swaps.SwapCandidates(ctx, eventID, msg.From.ID)
	if err != nil {
		text, ok := swapErrorText(err)
		if !ok {
			return err
		}

		return h.sendHTML(ctx, msg, text)
	}

	if len(view.Lines) < 2 {
		return h.sendHTML(ctx, msg, texts.SwapNobody)
	}

	text, kb := swapPeople(view, msg.From.ID, 0)

	return h.sendMenu(ctx, msg, text, kb)
}

func (h *Handler) showSwapConfirm(ctx context.Context, query telego.CallbackQuery, eventID, toID int64) error {
	view, err := h.swaps.SwapCandidates(ctx, eventID, query.From.ID)
	if err != nil {
		return h.swapStepFailed(ctx, query, err)
	}

	me, okMe := findLine(view, query.From.ID)
	other, okOther := findLine(view, toID)

	if !okMe || !okOther {
		return h.editMenu(ctx, query, texts.ParticipantNotFound, noButtons())
	}

	return h.editMenu(
		ctx, query, texts.SwapConfirm(view.Event.Title, me.Position, other.Name, other.Position),
		swapConfirmKeyboard(eventID, toID),
	)
}

func (h *Handler) sendSwap(ctx context.Context, query telego.CallbackQuery, eventID, toID int64) error {
	req, err := h.swaps.RequestSwap(ctx, eventID, query.From.ID, toID, query.Message.GetMessageID())
	if err != nil {
		return h.swapStepFailed(ctx, query, err)
	}

	return h.editMenu(ctx, query, texts.SwapSent(h.studentName(req.ToID)), swapCancelKeyboard(req.ID))
}

func (h *Handler) resolveSwap(ctx context.Context, query telego.CallbackQuery, a domain.Action) (string, error) {
	switch a.Kind {
	case domain.ActionSwapAccept:
		return h.acceptSwap(ctx, query, a.EventID)

	case domain.ActionSwapDecline:
		req, err := h.swaps.DeclineSwap(ctx, a.EventID, query.From.ID)
		if err != nil {
			return h.swapGone(ctx, query, err)
		}

		h.notify(ctx, req.FromID, texts.SwapDeclinedForInitiator(h.studentName(req.ToID), req.Event.Title))
		h.deleteMessage(ctx, req.FromID, req.FromMessageID)

		return "", h.editMenu(ctx, query, texts.SwapDeclined, noButtons())

	default:
		req, err := h.swaps.CancelSwap(ctx, a.EventID, query.From.ID)
		if err != nil {
			return h.swapGone(ctx, query, err)
		}

		h.deleteMessage(ctx, req.ToID, req.ToMessageID)

		return "", h.editMenu(ctx, query, texts.SwapCanceled, noButtons())
	}
}

func (h *Handler) acceptSwap(ctx context.Context, query telego.CallbackQuery, requestID int64) (string, error) {
	req, err := h.swaps.AcceptSwap(ctx, requestID, query.From.ID)

	switch {
	case errors.Is(err, domain.ErrSwapChanged),
		errors.Is(err, domain.ErrEventClosed),
		errors.Is(err, domain.ErrEventNotFound):
		h.notify(ctx, req.FromID, texts.SwapFailedForInitiator(req.Event.Title))
		h.deleteMessage(ctx, req.FromID, req.FromMessageID)

		return "", h.editMenu(ctx, query, texts.SwapChanged, noButtons())

	case err != nil:
		return h.swapGone(ctx, query, err)
	}

	h.notify(ctx, req.FromID, texts.SwapAcceptedForInitiator(h.studentName(req.ToID), req.Event.Title, req.ToPos))
	h.deleteMessage(ctx, req.FromID, req.FromMessageID)

	return "", h.editMenu(ctx, query, texts.SwapAcceptedForAddressee(req.FromPos), noButtons())
}

func (h *Handler) swapGone(ctx context.Context, query telego.CallbackQuery, err error) (string, error) {
	switch {
	case errors.Is(err, domain.ErrSwapForbidden):
		return texts.SwapNotYours, nil

	case errors.Is(err, domain.ErrSwapNotFound), errors.Is(err, domain.ErrSwapExpired):
		return "", h.editMenu(ctx, query, texts.SwapGone, noButtons())

	default:
		return "", err
	}
}

func (h *Handler) notify(ctx context.Context, userID int64, text string) {
	reqCtx, cancel := context.WithTimeout(ctx, h.apiTimeout)
	defer cancel()

	params := tu.Message(tu.ID(userID), text).WithParseMode(telego.ModeHTML)

	if _, err := h.bot.SendMessage(reqCtx, params); err != nil {
		h.log.WarnContext(
			ctx, "notify swap participant",
			slog.Int64("target_user_id", userID),
			slog.Any("err", err),
		)
	}
}

func (h *Handler) swapStepFailed(ctx context.Context, query telego.CallbackQuery, err error) error {
	text, ok := swapErrorText(err)
	if !ok {
		return err
	}

	return h.editMenu(ctx, query, text, noButtons())
}

func swapErrorText(err error) (string, bool) {
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

	case errors.Is(err, domain.ErrSwapBusy):
		return texts.SwapBusy, true

	default:
		return "", false
	}
}

func (h *Handler) studentName(userID int64) string {
	if name, ok := h.students.Name(userID); ok {
		return name
	}

	return fmt.Sprintf("id%d", userID)
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

func swapEventsKeyboard(events []domain.Event) *telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0, len(events))

	for _, e := range events {
		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					fmt.Sprintf("#%d %s", e.ID, e.Title), domain.Action{
						Kind:    domain.ActionSwapEvent,
						EventID: e.ID,
					},
				),
			),
		)
	}

	return tu.InlineKeyboard(rows...)
}

func swapPeople(view appqueue.View, selfID int64, requested int) (string, *telego.InlineKeyboardMarkup) {
	others := slices.DeleteFunc(
		slices.Clone(view.Lines), func(l appqueue.Line) bool {
			return l.UserID == selfID
		},
	)

	p := paginate(len(others), requested)
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+1)

	for _, line := range others[p.start:p.end] {
		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					fmt.Sprintf("%d. %s", line.Position, line.Name), domain.Action{
						Kind:    domain.ActionSwapPick,
						EventID: view.Event.ID,
						UserID:  line.UserID,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) domain.Action {
			return domain.Action{
				Kind:    domain.ActionSwapPage,
				EventID: view.Event.ID,
				Page:    index,
			}
		},
	)

	return texts.SwapChoosePerson(view.Event.ID, view.Event.Title, p.index, p.count), tu.InlineKeyboard(rows...)
}

func swapConfirmKeyboard(eventID, toID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonSwapSend, domain.Action{
					Kind:    domain.ActionSwapSend,
					EventID: eventID,
					UserID:  toID,
				},
			),
			button(
				texts.ButtonCancel, domain.Action{
					Kind:    domain.ActionSwapAbort,
					EventID: eventID,
				},
			),
		),
	)
}

func swapCancelKeyboard(requestID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonSwapCancel, domain.Action{
					Kind:    domain.ActionSwapCancel,
					EventID: requestID,
				},
			),
		),
	)
}
