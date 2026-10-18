package handler

import (
	"context"
	"errors"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func (h *Handler) HandleQueueCallback(ctx *th.Context, query telego.CallbackQuery) error {
	action, err := domain.ParseAction(query.Data)
	if err != nil {
		return h.answer(ctx, query, "")
	}

	var text string

	switch action.Kind {
	case domain.ActionJoin:
		text, err = h.join(ctx, action.EventID, query.From)

	case domain.ActionLeave:
		text, err = h.leave(ctx, action.EventID, query.From.ID)

	default:
		err = domain.ErrUnknownAction
	}

	if err != nil {
		_ = h.answer(ctx, query, texts.InternalError)

		return err
	}

	return h.answer(ctx, query, text)
}

func (h *Handler) join(ctx context.Context, eventID int64, user telego.User) (string, error) {
	res, err := h.queue.Join(ctx, eventID, user.ID, user.Username)

	switch {
	case errors.Is(err, domain.ErrEventClosed):
		return texts.QueueClosed, nil

	case errors.Is(err, domain.ErrEventNotFound):
		return texts.QueueNotFound, nil

	case err != nil:
		return "", err

	case res.Already:
		return texts.AlreadyInQueue(res.Position), nil

	default:
		return texts.JoinedQueue(res.Position), nil
	}
}

func (h *Handler) leave(ctx context.Context, eventID, userID int64) (string, error) {
	removed, err := h.queue.Leave(ctx, eventID, userID)

	switch {
	case errors.Is(err, domain.ErrEventClosed):
		return texts.QueueClosed, nil

	case errors.Is(err, domain.ErrEventNotFound):
		return texts.QueueNotFound, nil

	case err != nil:
		return "", err

	case !removed:
		return texts.NotInQueue, nil

	default:
		return texts.LeftQueue, nil
	}
}
