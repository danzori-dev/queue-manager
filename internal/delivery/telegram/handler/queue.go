package handler

import (
	"errors"
	"strings"
	"unicode"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
	"github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func (h *Handler) HandleSetTopic(ctx *th.Context, msg telego.Message) error {
	if !middleware.RoleFromContext(ctx).AtLeast(access.RoleAdmin) {
		return nil
	}

	if ok, err := h.studentsLoaded(ctx, msg); !ok {
		return err
	}

	replaced, err := h.queue.SetQueueTopic(ctx, msg.Chat.ID, msg.MessageThreadID)
	if err != nil {
		return err
	}

	h.deleteMessage(ctx, msg.Chat.ID, msg.MessageID)

	if replaced {
		return h.sendHTML(ctx, msg, texts.QueueTopicChanged)
	}

	return h.sendHTML(ctx, msg, texts.QueueTopicSet)
}

func (h *Handler) HandleNew(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.allowed(ctx, msg, access.RoleAdmin); !ok {
		return err
	}

	if ok, err := h.studentsLoaded(ctx, msg); !ok {
		return err
	}

	err := h.queue.CreateEvent(ctx, commandArgs(msg.Text))

	switch {
	case errors.Is(err, queue.ErrTargetNotSet):
		return h.sendHTML(ctx, msg, texts.QueueTopicNotSet)

	case errors.Is(err, queue.ErrEmptyTitle):
		return h.sendHTML(ctx, msg, texts.NewEventUsage)

	case errors.Is(err, queue.ErrTitleTooLong):
		return h.sendHTML(ctx, msg, texts.EventTitleTooLong)

	case err != nil:
		return err
	}

	return h.sendHTML(ctx, msg, texts.EventCreated)
}

func commandArgs(text string) string {
	i := strings.IndexFunc(text, unicode.IsSpace)
	if i < 0 {
		return ""
	}

	return strings.TrimSpace(text[i:])
}
