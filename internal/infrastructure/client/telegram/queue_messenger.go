package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	tu "github.com/mymmrac/telego/telegoutil"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type Renderer interface {
	Queue(view appqueue.View) (string, *telego.InlineKeyboardMarkup)
	Removal(removal appqueue.Removal) string
	SwapAsk(swap appqueue.Swap) (string, *telego.InlineKeyboardMarkup)
	InviteAsk(invite appqueue.Invite) (string, *telego.InlineKeyboardMarkup)
}

type QueueMessengerDeps struct {
	Bot        *telego.Bot
	Renderer   Renderer
	APITimeout time.Duration
	Log        *slog.Logger
}

type QueueMessenger struct {
	bot        *telego.Bot
	render     Renderer
	apiTimeout time.Duration
	log        *slog.Logger
}

func NewQueueMessenger(d QueueMessengerDeps) *QueueMessenger {
	return &QueueMessenger{
		bot:        d.Bot,
		render:     d.Renderer,
		apiTimeout: d.APITimeout,
		log:        d.Log,
	}
}

func (m *QueueMessenger) Publish(ctx context.Context, view appqueue.View) (int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, m.apiTimeout)
	defer cancel()

	text, kb := m.render.Queue(view)

	params := tu.Message(tu.ID(view.Event.ChatID), text).
		WithParseMode(telego.ModeHTML).
		WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true}).
		WithReplyMarkup(kb)

	if view.Event.ThreadID != 0 {
		params = params.WithMessageThreadID(view.Event.ThreadID)
	}

	msg, err := m.bot.SendMessage(reqCtx, params)
	if err != nil {
		return 0, fmt.Errorf("send queue message: %w", err)
	}

	return msg.MessageID, nil
}

func (m *QueueMessenger) Edit(ctx context.Context, view appqueue.View) (time.Duration, error) {
	reqCtx, cancel := context.WithTimeout(ctx, m.apiTimeout)
	defer cancel()

	text, kb := m.render.Queue(view)

	params := tu.EditMessageText(tu.ID(view.Event.ChatID), view.Event.MessageID, text).
		WithParseMode(telego.ModeHTML).
		WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true}).
		WithReplyMarkup(kb)

	if _, err := m.bot.EditMessageText(reqCtx, params); err != nil {
		apiErr, ok := errors.AsType[*ta.Error](err)

		switch {
		case ok && apiErr.Parameters != nil && apiErr.Parameters.RetryAfter > 0:
			return time.Duration(apiErr.Parameters.RetryAfter) * time.Second, nil

		case ok && strings.Contains(apiErr.Description, "message is not modified"):
			return 0, nil

		case ok && strings.Contains(apiErr.Description, "message to edit not found"):
			m.log.WarnContext(
				ctx, "message to edit not found, republishing",
				slog.Int64("event_id", view.Event.ID),
			)

			return 0, domain.ErrMessageNotFound
		}

		return 0, fmt.Errorf("edit queue message: %w", err)
	}

	return 0, nil
}

func (m *QueueMessenger) Delete(ctx context.Context, event domain.Event) {
	if event.MessageID == 0 {
		return
	}

	reqCtx, cancel := context.WithTimeout(ctx, m.apiTimeout)
	defer cancel()

	err := m.bot.DeleteMessage(
		reqCtx, &telego.DeleteMessageParams{
			ChatID:    tu.ID(event.ChatID),
			MessageID: event.MessageID,
		},
	)
	if err != nil {
		m.log.WarnContext(
			ctx, "delete queue message",
			slog.Int64("event_id", event.ID),
			slog.Any("err", err),
		)
	}
}

func (m *QueueMessenger) AnnounceRemoval(ctx context.Context, removal appqueue.Removal) {
	reqCtx, cancel := context.WithTimeout(ctx, m.apiTimeout)
	defer cancel()

	event := removal.Event

	params := tu.Message(tu.ID(event.ChatID), m.render.Removal(removal)).
		WithParseMode(telego.ModeHTML)

	if event.ThreadID != 0 {
		params = params.WithMessageThreadID(event.ThreadID)
	}

	if event.MessageID != 0 {
		params = params.WithReplyParameters(
			&telego.ReplyParameters{
				MessageID:                event.MessageID,
				AllowSendingWithoutReply: true,
			},
		)
	}

	if _, err := m.bot.SendMessage(reqCtx, params); err != nil {
		m.log.WarnContext(
			ctx, "announce participant removal",
			slog.Int64("event_id", event.ID),
			slog.Int64("target_user_id", removal.UserID),
			slog.Any("err", err),
		)
	}
}

func (m *QueueMessenger) AskSwap(ctx context.Context, swap appqueue.Swap) (int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, m.apiTimeout)
	defer cancel()

	text, kb := m.render.SwapAsk(swap)

	params := tu.Message(tu.ID(swap.ToID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(kb)

	msg, err := m.bot.SendMessage(reqCtx, params)
	if err != nil {
		if apiErr, ok := errors.AsType[*ta.Error](err); ok && apiErr.ErrorCode == http.StatusForbidden {
			return 0, domain.ErrUserUnreachable
		}

		return 0, fmt.Errorf("send swap request: %w", err)
	}

	return msg.MessageID, nil
}

func (m *QueueMessenger) AskInvite(ctx context.Context, invite appqueue.Invite) (int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, m.apiTimeout)
	defer cancel()

	text, kb := m.render.InviteAsk(invite)

	params := tu.Message(tu.ID(invite.ToID), text).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(kb)

	msg, err := m.bot.SendMessage(reqCtx, params)
	if err != nil {
		if apiErr, ok := errors.AsType[*ta.Error](err); ok && apiErr.ErrorCode == http.StatusForbidden {
			return 0, domain.ErrUserUnreachable
		}

		return 0, fmt.Errorf("send invite: %w", err)
	}

	return msg.MessageID, nil
}
