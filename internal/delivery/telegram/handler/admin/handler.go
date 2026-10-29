package admin

import (
	"context"
	"errors"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/command"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/reply"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

type Handler struct {
	reply    *reply.Replier
	admins   AdminService
	students StudentService
}

func New(replier *reply.Replier, admins AdminService, students StudentService) *Handler {
	return &Handler{
		reply:    replier,
		admins:   admins,
		students: students,
	}
}

func (h *Handler) Add(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	if ok, err := h.reply.RequireStudents(ctx, msg, h.students.Count()); !ok {
		return err
	}

	targetID, ok := command.UserID(msg.Text)
	if !ok {
		return h.reply.SendHTML(ctx, msg, texts.AddAdminUsage)
	}

	if _, ok := h.students.Name(targetID); !ok {
		return h.reply.SendHTML(ctx, msg, texts.NotAStudent)
	}

	if err := h.admins.Add(ctx, targetID); err != nil {
		return h.replyError(ctx, msg, err)
	}

	return h.reply.SendHTML(ctx, msg, texts.AdminAdded(h.userLabel(targetID)))
}

func (h *Handler) Remove(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	if ok, err := h.reply.RequireStudents(ctx, msg, h.students.Count()); !ok {
		return err
	}

	targetID, ok := command.UserID(msg.Text)
	if !ok {
		return h.reply.SendHTML(ctx, msg, texts.RmAdminUsage)
	}

	if err := h.admins.Remove(ctx, targetID); err != nil {
		return h.replyError(ctx, msg, err)
	}

	return h.reply.SendHTML(ctx, msg, texts.AdminRemoved(h.userLabel(targetID)))
}

func (h *Handler) List(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	if ok, err := h.reply.RequireStudents(ctx, msg, h.students.Count()); !ok {
		return err
	}

	ids := h.admins.List()

	labels := make([]string, 0, len(ids))
	for _, id := range ids {
		labels = append(labels, h.userLabel(id))
	}

	return h.reply.SendHTML(ctx, msg, texts.AdminList(labels))
}

func (h *Handler) replyError(ctx context.Context, msg telego.Message, err error) error {
	switch {
	case errors.Is(err, access.ErrCannotAddSelf):
		return h.reply.SendHTML(ctx, msg, texts.CannotAddSelf)

	case errors.Is(err, access.ErrCannotRemoveOwner):
		return h.reply.SendHTML(ctx, msg, texts.CannotRemoveOwner)

	case errors.Is(err, access.ErrAdminAlreadyExists):
		return h.reply.SendHTML(ctx, msg, texts.AdminAlreadyExists)

	case errors.Is(err, access.ErrAdminNotFound):
		return h.reply.SendHTML(ctx, msg, texts.AdminNotFound)

	default:
		return err
	}
}

func (h *Handler) userLabel(userID int64) string {
	name, _ := h.students.Name(userID)

	return texts.UserLabel(userID, name)
}
