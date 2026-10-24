package reply

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/server/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

type Replier struct {
	bot        *telego.Bot
	apiTimeout time.Duration
	log        *slog.Logger
}

func New(bot *telego.Bot, apiTimeout time.Duration, log *slog.Logger) *Replier {
	return &Replier{
		bot:        bot,
		apiTimeout: apiTimeout,
		log:        log,
	}
}

func (r *Replier) SendHTML(ctx context.Context, msg telego.Message, text string) error {
	return r.SendMenu(ctx, msg, text, nil)
}

func (r *Replier) SendMenu(ctx context.Context, msg telego.Message, text string, kb *telego.InlineKeyboardMarkup) error {
	reqCtx, cancel := context.WithTimeout(ctx, r.apiTimeout)
	defer cancel()

	params := tu.Message(tu.ID(msg.Chat.ID), text).
		WithParseMode(telego.ModeHTML).
		WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true})

	if kb != nil {
		params = params.WithReplyMarkup(kb)
	}

	if msg.MessageThreadID != 0 {
		params = params.WithMessageThreadID(msg.MessageThreadID)
	}

	if _, err := r.bot.SendMessage(reqCtx, params); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func (r *Replier) EditMenu(ctx context.Context, query telego.CallbackQuery, text string, kb *telego.InlineKeyboardMarkup) error {
	reqCtx, cancel := context.WithTimeout(ctx, r.apiTimeout)
	defer cancel()

	params := tu.EditMessageText(tu.ID(query.Message.GetChat().ID), query.Message.GetMessageID(), text).
		WithParseMode(telego.ModeHTML).
		WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true})

	if kb != nil {
		params = params.WithReplyMarkup(kb)
	}

	if _, err := r.bot.EditMessageText(reqCtx, params); err != nil {
		apiErr, ok := errors.AsType[*ta.Error](err)
		if ok && strings.Contains(apiErr.Description, "message is not modified") {
			return nil
		}

		return fmt.Errorf("edit menu: %w", err)
	}

	return nil
}

func (r *Replier) Answer(ctx context.Context, query telego.CallbackQuery, text string) error {
	reqCtx, cancel := context.WithTimeout(ctx, r.apiTimeout)
	defer cancel()

	params := tu.CallbackQuery(query.ID)

	if text != "" {
		params = params.WithText(text)
	}

	if err := r.bot.AnswerCallbackQuery(reqCtx, params); err != nil {
		return fmt.Errorf("answer callback: %w", err)
	}

	return nil
}

func (r *Replier) DeleteMessage(ctx context.Context, chatID int64, messageID int) {
	if messageID == 0 {
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx, r.apiTimeout)
	defer cancel()

	err := r.bot.DeleteMessage(
		reqCtx, &telego.DeleteMessageParams{
			ChatID:    tu.ID(chatID),
			MessageID: messageID,
		},
	)
	if err != nil {
		r.log.DebugContext(
			ctx, "delete message",
			slog.Any("err", err),
		)
	}
}

func (r *Replier) Notify(ctx context.Context, userID int64, text string) {
	reqCtx, cancel := context.WithTimeout(ctx, r.apiTimeout)
	defer cancel()

	params := tu.Message(tu.ID(userID), text).WithParseMode(telego.ModeHTML)

	if _, err := r.bot.SendMessage(reqCtx, params); err != nil {
		r.log.WarnContext(
			ctx, "notify user",
			slog.Int64("target_user_id", userID),
			slog.Any("err", err),
		)
	}
}

func (r *Replier) Allowed(ctx context.Context, msg telego.Message, minRole access.Role) (bool, error) {
	if middleware.RoleFromContext(ctx).AtLeast(minRole) {
		return true, nil
	}

	return false, r.SendHTML(ctx, msg, texts.AccessDenied)
}

func (r *Replier) RequireStudents(ctx context.Context, msg telego.Message, count int) (bool, error) {
	if count > 0 {
		return true, nil
	}

	return false, r.SendHTML(ctx, msg, texts.StudentsNotLoaded)
}
