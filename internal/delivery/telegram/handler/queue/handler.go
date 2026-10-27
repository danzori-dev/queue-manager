package queue

import (
	"context"
	"errors"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/command"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/confirm"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/reply"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/server/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type Handler struct {
	reply    *reply.Replier
	queues   QueueService
	students StudentService
	confirms *confirm.DoublePress
}

func New(replier *reply.Replier, queues QueueService, students StudentService) *Handler {
	return &Handler{
		reply:    replier,
		queues:   queues,
		students: students,
		confirms: confirm.New(),
	}
}

func (h *Handler) SetTopic(ctx *th.Context, msg telego.Message) error {
	if !middleware.RoleFromContext(ctx).AtLeast(access.RoleAdmin) {
		return nil
	}

	if ok, err := h.reply.RequireStudents(ctx, msg, h.students.Count()); !ok {
		return err
	}

	replaced, err := h.queues.SetQueueTopic(ctx, msg.Chat.ID, msg.MessageThreadID)
	if err != nil {
		return err
	}

	h.reply.DeleteMessage(ctx, msg.Chat.ID, msg.MessageID)

	if replaced {
		return h.reply.SendHTML(ctx, msg, texts.QueueTopicChanged)
	}

	return h.reply.SendHTML(ctx, msg, texts.QueueTopicSet)
}

func (h *Handler) Create(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleAdmin); !ok {
		return err
	}

	if ok, err := h.reply.RequireStudents(ctx, msg, h.students.Count()); !ok {
		return err
	}

	err := h.queues.CreateEvent(ctx, msg.From.ID, command.Args(msg.Text))

	switch {
	case errors.Is(err, domain.ErrTargetNotSet):
		return h.reply.SendHTML(ctx, msg, texts.QueueTopicNotSet)

	case errors.Is(err, domain.ErrEmptyTitle):
		return h.reply.SendHTML(ctx, msg, texts.NewEventUsage)

	case errors.Is(err, domain.ErrTitleTooLong):
		return h.reply.SendHTML(ctx, msg, texts.EventTitleTooLong)

	case err != nil:
		return err
	}

	return h.reply.SendHTML(ctx, msg, texts.EventCreated)
}

func (h *Handler) Callback(ctx *th.Context, query telego.CallbackQuery) error {
	action, err := callback.ParseAction(query.Data)
	if err != nil {
		return h.reply.Answer(ctx, query, "")
	}

	var text string

	switch action.Kind {
	case callback.ActionJoin:
		text, err = h.join(ctx, action.EventID, query.From)

	case callback.ActionLeave:
		text, err = h.leave(ctx, action.EventID, query.From.ID)

	case callback.ActionSkip:
		text, err = h.skip(ctx, action.EventID, query.From.ID)

	default:
		err = callback.ErrUnknownAction
	}

	if err != nil {
		_ = h.reply.Answer(ctx, query, texts.InternalError)

		return err
	}

	return h.reply.Answer(ctx, query, text)
}

func (h *Handler) join(ctx context.Context, eventID int64, user telego.User) (string, error) {
	res, err := h.queues.Join(ctx, eventID, user.ID, user.Username)
	if err != nil {
		return errorText(err)
	}

	if res.Already {
		return texts.AlreadyInQueue(res.Position), nil
	}

	return texts.JoinedQueue(res.Position), nil
}

func (h *Handler) leave(ctx context.Context, eventID, userID int64) (string, error) {
	removed, err := h.queues.Leave(ctx, eventID, userID)
	if err != nil {
		return errorText(err)
	}

	if !removed {
		return texts.NotInQueue, nil
	}

	return texts.LeftQueue, nil
}

func (h *Handler) skip(ctx context.Context, eventID, userID int64) (string, error) {
	if !h.confirms.Confirm(callback.ActionSkip, eventID, userID, userID) {
		return texts.SkipPressAgain, nil
	}

	position, err := h.queues.Skip(ctx, eventID, userID)
	if err != nil {
		return errorText(err)
	}

	return texts.Skipped(position), nil
}

func errorText(err error) (string, error) {
	switch {
	case errors.Is(err, domain.ErrEventClosed):
		return texts.QueueClosed, nil

	case errors.Is(err, domain.ErrEventNotFound):
		return texts.QueueNotFound, nil

	case errors.Is(err, domain.ErrNotInQueue):
		return texts.NotInQueue, nil

	case errors.Is(err, domain.ErrLastInQueue):
		return texts.AlreadyLast, nil

	default:
		return "", err
	}
}
