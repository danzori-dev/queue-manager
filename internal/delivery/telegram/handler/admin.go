package handler

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

func (h *Handler) HandleAddAdmin(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	if ok, err := h.studentsLoaded(ctx, msg); !ok {
		return err
	}

	targetID, ok := parseUserIDArg(msg.Text)
	if !ok {
		return h.sendHTML(ctx, msg, texts.AddAdminUsage)
	}

	if _, ok := h.students.Name(targetID); !ok {
		return h.sendHTML(ctx, msg, texts.NotAStudent)
	}

	if err := h.admin.Add(ctx, targetID); err != nil {
		return h.replyAdminError(ctx, msg, err)
	}

	return h.sendHTML(ctx, msg, texts.AdminAdded(h.userLabel(targetID)))
}

func (h *Handler) HandleRmAdmin(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	if ok, err := h.studentsLoaded(ctx, msg); !ok {
		return err
	}

	targetID, ok := parseUserIDArg(msg.Text)
	if !ok {
		return h.sendHTML(ctx, msg, texts.RmAdminUsage)
	}

	if err := h.admin.Remove(ctx, targetID); err != nil {
		return h.replyAdminError(ctx, msg, err)
	}

	return h.sendHTML(ctx, msg, texts.AdminRemoved(h.userLabel(targetID)))
}

func (h *Handler) HandleAdmins(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	if ok, err := h.studentsLoaded(ctx, msg); !ok {
		return err
	}

	ids, err := h.admin.List(ctx)
	if err != nil {
		return err
	}

	labels := make([]string, 0, len(ids))
	for _, id := range ids {
		labels = append(labels, h.userLabel(id))
	}

	return h.sendHTML(ctx, msg, texts.AdminList(labels))
}

func (h *Handler) replyAdminError(ctx context.Context, msg telego.Message, err error) error {
	switch {
	case errors.Is(err, access.ErrCannotAddSelf):
		return h.sendHTML(ctx, msg, texts.CannotAddSelf)

	case errors.Is(err, access.ErrCannotRemoveOwner):
		return h.sendHTML(ctx, msg, texts.CannotRemoveOwner)

	case errors.Is(err, access.ErrAdminAlreadyExists):
		return h.sendHTML(ctx, msg, texts.AdminAlreadyExists)

	case errors.Is(err, access.ErrAdminNotFound):
		return h.sendHTML(ctx, msg, texts.AdminNotFound)

	default:
		return err
	}
}

func parseUserIDArg(text string) (int64, bool) {
	_, _, args := tu.ParseCommand(text)
	if len(args) != 1 {
		return 0, false
	}

	id, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}
