package handler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

type Deps struct {
	Bot        *telego.Bot
	Admin      AdminService
	Students   StudentService
	Queue      QueueService
	APITimeout time.Duration
	Log        *slog.Logger
}

type Handler struct {
	bot        *telego.Bot
	admin      AdminService
	students   StudentService
	queue      QueueService
	apiTimeout time.Duration
	log        *slog.Logger
}

func New(d Deps) *Handler {
	return &Handler{
		bot:        d.Bot,
		admin:      d.Admin,
		students:   d.Students,
		queue:      d.Queue,
		apiTimeout: d.APITimeout,
		log:        d.Log,
	}
}

func (h *Handler) sendHTML(ctx context.Context, msg telego.Message, text string) error {
	reqCtx, cancel := context.WithTimeout(ctx, h.apiTimeout)
	defer cancel()

	params := tu.Message(tu.ID(msg.Chat.ID), text).
		WithParseMode(telego.ModeHTML).
		WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true})

	if msg.MessageThreadID != 0 {
		params = params.WithMessageThreadID(msg.MessageThreadID)
	}

	if _, err := h.bot.SendMessage(reqCtx, params); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func (h *Handler) answer(ctx context.Context, query telego.CallbackQuery, text string) error {
	reqCtx, cancel := context.WithTimeout(ctx, h.apiTimeout)
	defer cancel()

	params := tu.CallbackQuery(query.ID)

	if text != "" {
		params = params.WithText(text)
	}

	if err := h.bot.AnswerCallbackQuery(reqCtx, params); err != nil {
		return fmt.Errorf("answer callback: %w", err)
	}

	return nil
}

func (h *Handler) deleteMessage(ctx context.Context, msg telego.Message) {
	reqCtx, cancel := context.WithTimeout(ctx, h.apiTimeout)
	defer cancel()

	err := h.bot.DeleteMessage(
		reqCtx, &telego.DeleteMessageParams{
			ChatID:    tu.ID(msg.Chat.ID),
			MessageID: msg.MessageID,
		},
	)
	if err != nil {
		h.log.DebugContext(
			ctx, "delete command message",
			slog.Any("err", err),
		)
	}
}

func (h *Handler) allowed(ctx context.Context, msg telego.Message, minRole access.Role) (bool, error) {
	if middleware.RoleFromContext(ctx).AtLeast(minRole) {
		return true, nil
	}

	return false, h.sendHTML(ctx, msg, texts.AccessDenied)
}

func (h *Handler) studentsLoaded(ctx context.Context, msg telego.Message) (bool, error) {
	if h.students.Count() > 0 {
		return true, nil
	}

	return false, h.sendHTML(ctx, msg, texts.StudentsNotLoaded)
}

func (h *Handler) userLabel(userID int64) string {
	name, _ := h.students.Name(userID)

	return texts.UserLabel(userID, name)
}
