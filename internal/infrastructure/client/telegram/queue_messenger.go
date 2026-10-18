package telegram

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	tu "github.com/mymmrac/telego/telegoutil"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type QueueMessengerDeps struct {
	Bot        *telego.Bot
	APITimeout time.Duration
	Log        *slog.Logger
}

type QueueMessenger struct {
	bot        *telego.Bot
	apiTimeout time.Duration
	log        *slog.Logger
}

func NewQueueMessenger(d QueueMessengerDeps) *QueueMessenger {
	return &QueueMessenger{
		bot:        d.Bot,
		apiTimeout: d.APITimeout,
		log:        d.Log,
	}
}

func (m *QueueMessenger) Publish(ctx context.Context, view appqueue.View) (int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, m.apiTimeout)
	defer cancel()

	params := tu.Message(tu.ID(view.Event.ChatID), renderQueue(view)).
		WithParseMode(telego.ModeHTML).
		WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true}).
		WithReplyMarkup(queueKeyboard(view))

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

	params := tu.EditMessageText(tu.ID(view.Event.ChatID), view.Event.MessageID, renderQueue(view)).
		WithParseMode(telego.ModeHTML).
		WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true}).
		WithReplyMarkup(queueKeyboard(view))

	if _, err := m.bot.EditMessageText(reqCtx, params); err != nil {
		apiErr, ok := errors.AsType[*ta.Error](err)

		switch {
		case ok && apiErr.Parameters != nil && apiErr.Parameters.RetryAfter > 0:
			return time.Duration(apiErr.Parameters.RetryAfter) * time.Second, nil

		case ok && strings.Contains(apiErr.Description, "message is not modified"):
			return 0, nil
		}

		return 0, fmt.Errorf("edit queue message: %w", err)
	}

	return 0, nil
}

func renderQueue(view appqueue.View) string {
	var b strings.Builder

	b.WriteString("<b>" + html.EscapeString(view.Event.Title) + "</b>\n")

	if len(view.Lines) == 0 {
		b.WriteString("\nВ очереди никого нет.")

		return b.String()
	}

	for _, line := range view.Lines {
		fmt.Fprintf(&b, "\n%d. %s", line.Position, html.EscapeString(line.Name))

		if line.Username != "" {
			fmt.Fprintf(&b, " (@%s)", line.Username)
		}
	}

	fmt.Fprintf(&b, "\n\nВ очереди: %d", len(view.Lines))

	return b.String()
}

func queueKeyboard(view appqueue.View) *telego.InlineKeyboardMarkup {
	join := domain.Action{
		Kind:    domain.ActionJoin,
		EventID: view.Event.ID,
	}
	leave := domain.Action{
		Kind:    domain.ActionLeave,
		EventID: view.Event.ID,
	}

	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Встать в очередь").WithCallbackData(join.String()),
			tu.InlineKeyboardButton("Выйти").WithCallbackData(leave.String()),
		),
	)
}
