package telegram

import (
	"context"
	"errors"
	"fmt"
	"html"
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

type QueueMessengerDeps struct {
	Bot         *telego.Bot
	BotUsername string
	APITimeout  time.Duration
	Log         *slog.Logger
}

type QueueMessenger struct {
	bot         *telego.Bot
	botUsername string
	apiTimeout  time.Duration
	log         *slog.Logger
}

func NewQueueMessenger(d QueueMessengerDeps) *QueueMessenger {
	return &QueueMessenger{
		bot:         d.Bot,
		botUsername: d.BotUsername,
		apiTimeout:  d.APITimeout,
		log:         d.Log,
	}
}

func (m *QueueMessenger) Publish(ctx context.Context, view appqueue.View) (int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, m.apiTimeout)
	defer cancel()

	params := tu.Message(tu.ID(view.Event.ChatID), renderQueue(view)).
		WithParseMode(telego.ModeHTML).
		WithLinkPreviewOptions(&telego.LinkPreviewOptions{IsDisabled: true}).
		WithReplyMarkup(m.queueKeyboard(view))

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
		WithReplyMarkup(m.queueKeyboard(view))

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

	params := tu.Message(tu.ID(event.ChatID), renderRemoval(removal)).
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

func (m *QueueMessenger) AskSwap(ctx context.Context, ask appqueue.SwapAsk) (int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, m.apiTimeout)
	defer cancel()

	params := tu.Message(tu.ID(ask.Request.ToID), renderSwapAsk(ask)).
		WithParseMode(telego.ModeHTML).
		WithReplyMarkup(swapKeyboard(ask.Request.ID))

	msg, err := m.bot.SendMessage(reqCtx, params)
	if err != nil {
		if apiErr, ok := errors.AsType[*ta.Error](err); ok && apiErr.ErrorCode == http.StatusForbidden {
			return 0, domain.ErrSwapUnreachable
		}

		return 0, fmt.Errorf("send swap request: %w", err)
	}

	return msg.MessageID, nil
}

func renderQueue(view appqueue.View) string {
	var b strings.Builder

	fmt.Fprintf(&b, "<b>#%d %s</b>\n", view.Event.ID, html.EscapeString(view.Event.Title))

	if view.Event.Status == domain.StatusClosed {
		b.WriteString("🔒 Запись закрыта.\n")
	}

	if len(view.Lines) == 0 {
		b.WriteString("\nВ очереди никого нет.")

		return b.String()
	}

	for _, line := range view.Lines {
		fmt.Fprintf(&b, "\n%d. %s", line.Position, html.EscapeString(line.Name))

		if line.UserName != "" {
			fmt.Fprintf(&b, " (@%s)", line.UserName)
		}
	}

	fmt.Fprintf(&b, "\n\nВ очереди: %d", len(view.Lines))

	return b.String()
}

func renderRemoval(r appqueue.Removal) string {
	actor := "Администратор"

	if r.ActorName != "" {
		actor += " " + html.EscapeString(r.ActorName)
	}

	return fmt.Sprintf(
		`%s убрал <a href="tg://user?id=%d">%s</a> из очереди «%s».`,
		actor, r.UserID, html.EscapeString(r.UserName), html.EscapeString(r.Event.Title),
	)
}

func renderSwapAsk(ask appqueue.SwapAsk) string {
	r := ask.Request

	return fmt.Sprintf(
		"%s предлагает поменяться с тобой местами в очереди «%s».\n\n"+
			"Твое место - %d\nМесто %s - %d\n\n"+
			"Запрос действует %d минут.",
		html.EscapeString(ask.FromName), html.EscapeString(r.Event.Title),
		r.ToPos, html.EscapeString(ask.FromName), r.FromPos, int(domain.SwapTTL.Minutes()),
	)
}

func (m *QueueMessenger) queueKeyboard(view appqueue.View) *telego.InlineKeyboardMarkup {
	if view.Event.Status == domain.StatusClosed {
		return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{}}
	}

	join := domain.Action{
		Kind:    domain.ActionJoin,
		EventID: view.Event.ID,
	}
	leave := domain.Action{
		Kind:    domain.ActionLeave,
		EventID: view.Event.ID,
	}
	skip := domain.Action{
		Kind:    domain.ActionSkip,
		EventID: view.Event.ID,
	}

	swapURL := fmt.Sprintf("https://t.me/%s?start=swap_%d", m.botUsername, view.Event.ID)

	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Встать в очередь").WithCallbackData(join.String()),
			tu.InlineKeyboardButton("Выйти").WithCallbackData(leave.String()),
		),
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Пропустить место").WithCallbackData(skip.String()),
			tu.InlineKeyboardButton("Поменяться местами").WithURL(swapURL),
		),
	)
}

func swapKeyboard(requestID int64) *telego.InlineKeyboardMarkup {
	accept := domain.Action{
		Kind:    domain.ActionSwapAccept,
		EventID: requestID,
	}
	decline := domain.Action{
		Kind:    domain.ActionSwapDecline,
		EventID: requestID,
	}

	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			tu.InlineKeyboardButton("Принять").WithCallbackData(accept.String()),
			tu.InlineKeyboardButton("Отклонить").WithCallbackData(decline.String()),
		),
	)
}
