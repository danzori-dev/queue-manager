package invite

import (
	"context"
	"errors"
	"slices"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/keyboard"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/reply"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
	"github.com/danzori-dev/queue-manager/internal/domain/student"
)

type Handler struct {
	reply   *reply.Replier
	invites InviteService
}

func New(replier *reply.Replier, invites InviteService) *Handler {
	return &Handler{
		reply:   replier,
		invites: invites,
	}
}

func (h *Handler) Invite(ctx *th.Context, msg telego.Message) error {
	events, err := h.invites.InviteTargets(ctx, msg.From.ID)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return h.reply.SendHTML(ctx, msg, texts.InviteNotQueues)
	}

	return h.reply.SendMenu(ctx, msg, texts.InviteChooseQueue, keyboard.InviteEvents(events))
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
	c, err := h.invites.InviteCandidates(ctx, eventID, msg.From.ID)
	if err != nil {
		text, ok := errorText(err)
		if !ok {
			return err
		}

		return h.reply.SendHTML(ctx, msg, text)
	}

	if len(c.Students) == 0 {
		return h.reply.SendHTML(ctx, msg, texts.InviteNobody)
	}

	kb, p := keyboard.InvitePeople(c, 0)

	return h.reply.SendMenu(ctx, msg, texts.InviteChoosePerson(c.Event.ID, c.Event.Title, c.Position, p.Index, p.Count), kb)
}

func (h *Handler) applyAction(ctx context.Context, query telego.CallbackQuery, a callback.Action) (string, error) {
	switch a.Kind {
	case callback.ActionInviteEvent, callback.ActionInvitePage:
		return "", h.showCandidates(ctx, query, a.EventID, a.Page)

	case callback.ActionInvitePick:
		return "", h.showConfirm(ctx, query, a.EventID, a.UserID)

	case callback.ActionInviteSend:
		return "", h.send(ctx, query, a.EventID, a.UserID)

	case callback.ActionInviteAbort:
		return "", h.reply.EditMenu(ctx, query, texts.InviteAborted, keyboard.None())

	case callback.ActionInviteAccept:
		return h.accept(ctx, query, a.EventID)

	case callback.ActionInviteDecline:
		return h.decline(ctx, query, a.EventID)

	case callback.ActionInviteCancel:
		return h.cancel(ctx, query, a.EventID)

	default:
		return "", callback.ErrUnknownAction
	}
}

func (h *Handler) showCandidates(ctx context.Context, query telego.CallbackQuery, eventID int64, page int) error {
	c, err := h.invites.InviteCandidates(ctx, eventID, query.From.ID)
	if err != nil {
		return h.stepFailed(ctx, query, err)
	}

	if len(c.Students) == 0 {
		return h.reply.EditMenu(ctx, query, texts.InviteNobody, keyboard.None())
	}

	kb, p := keyboard.InvitePeople(c, page)

	return h.reply.EditMenu(ctx, query, texts.InviteChoosePerson(c.Event.ID, c.Event.Title, c.Position, p.Index, p.Count), kb)
}

func (h *Handler) showConfirm(ctx context.Context, query telego.CallbackQuery, eventID, toID int64) error {
	c, err := h.invites.InviteCandidates(ctx, eventID, query.From.ID)
	if err != nil {
		return h.stepFailed(ctx, query, err)
	}

	i := slices.IndexFunc(
		c.Students, func(st student.Student) bool {
			return st.ID == toID
		},
	)
	if i < 0 {
		return h.reply.EditMenu(ctx, query, texts.InviteUnavailable, keyboard.None())
	}

	return h.reply.EditMenu(
		ctx, query, texts.InviteConfirm(c.Students[i].Name, c.Event.Title, c.Position),
		keyboard.InviteConfirm(eventID, toID),
	)
}

func (h *Handler) send(ctx context.Context, query telego.CallbackQuery, eventID, toID int64) error {
	inv, err := h.invites.RequestInvite(ctx, eventID, query.From.ID, toID, query.Message.GetMessageID())
	if err != nil {
		return h.stepFailed(ctx, query, err)
	}

	return h.reply.EditMenu(ctx, query, texts.InviteSent(inv.ToName), keyboard.InviteCancel(inv.ID))
}

func (h *Handler) accept(ctx context.Context, query telego.CallbackQuery, requestID int64) (string, error) {
	inv, err := h.invites.AcceptInvite(ctx, requestID, query.From.ID, query.From.Username)

	switch {
	case errors.Is(err, domain.ErrInviteChanged):
		h.reply.DeleteMessage(ctx, inv.FromID, inv.FromMessageID)

		return "", h.reply.EditMenu(ctx, query, texts.InviteChanged, keyboard.None())

	case errors.Is(err, domain.ErrEventClosed),
		errors.Is(err, domain.ErrEventNotFound),
		errors.Is(err, domain.ErrPlaceFull),
		errors.Is(err, domain.ErrAlreadyOnPlace):
		text, _ := errorText(err)

		h.reply.Notify(ctx, inv.FromID, texts.InviteFailedForInitiator(inv.ToName, inv.Event.Title))
		h.reply.DeleteMessage(ctx, inv.FromID, inv.FromMessageID)

		return "", h.reply.EditMenu(ctx, query, text, keyboard.None())

	case err != nil:
		return h.gone(ctx, query, err)
	}

	h.reply.Notify(ctx, inv.FromID, texts.InviteAcceptedForInitiator(inv.ToName, inv.Event.Title, inv.Position))
	h.reply.DeleteMessage(ctx, inv.FromID, inv.FromMessageID)

	return "", h.reply.EditMenu(ctx, query, texts.InviteAcceptedForAddressee(inv.Event.Title, inv.Position), keyboard.None())
}

func (h *Handler) decline(ctx context.Context, query telego.CallbackQuery, requestID int64) (string, error) {
	inv, err := h.invites.DeclineInvite(ctx, requestID, query.From.ID)
	if err != nil {
		return h.gone(ctx, query, err)
	}

	h.reply.Notify(ctx, inv.FromID, texts.InviteDeclinedForInitiator(inv.ToName, inv.Event.Title))
	h.reply.DeleteMessage(ctx, inv.FromID, inv.FromMessageID)

	return "", h.reply.EditMenu(ctx, query, texts.InviteDeclined, keyboard.None())
}

func (h *Handler) cancel(ctx context.Context, query telego.CallbackQuery, requestID int64) (string, error) {
	inv, err := h.invites.CancelInvite(ctx, requestID, query.From.ID)
	if err != nil {
		return h.gone(ctx, query, err)
	}

	h.reply.DeleteMessage(ctx, inv.ToID, inv.ToMessageID)

	return "", h.reply.EditMenu(ctx, query, texts.InviteCanceled, keyboard.None())
}

func (h *Handler) gone(ctx context.Context, query telego.CallbackQuery, err error) (string, error) {
	switch {
	case errors.Is(err, domain.ErrInviteForbidden):
		return texts.InviteNotYours, nil

	case errors.Is(err, domain.ErrInviteNotFound), errors.Is(err, domain.ErrInviteExpired):
		return "", h.reply.EditMenu(ctx, query, texts.InviteGone, keyboard.None())

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
		return texts.InviteNotInQueue, true

	case errors.Is(err, domain.ErrUserUnreachable):
		return texts.UserUnreachable, true

	case errors.Is(err, domain.ErrInviteSelf):
		return texts.InviteSelf, true

	case errors.Is(err, domain.ErrAlreadyOnPlace):
		return texts.AlreadyOnPlace, true

	case errors.Is(err, domain.ErrPlaceFull):
		return texts.PlaceFull, true

	case errors.Is(err, domain.ErrInviteBusy):
		return texts.InviteBusy, true

	default:
		return "", false
	}
}
