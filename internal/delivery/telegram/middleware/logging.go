package middleware

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/application/logctx"
)

func Logging(log *slog.Logger) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		start := time.Now()
		stdCtx := injectContext(ctx.Context(), update)
		ctx = ctx.WithContext(stdCtx)

		err := ctx.Next(update)
		durationMs := time.Since(start).Milliseconds()

		if err != nil {
			log.ErrorContext(
				ctx.Context(), "update handling failed",
				slog.Int64("duration_ms", durationMs),
				slog.Any("err", err),
			)

			return nil
		}

		log.InfoContext(
			ctx.Context(), "update handled",
			slog.Int64("duration_ms", durationMs),
		)

		return nil
	}
}

func injectContext(ctx context.Context, update telego.Update) context.Context {
	ctx = logctx.WithUpdateID(ctx, update.UpdateID)

	switch {
	case update.Message != nil:
		ctx = logctx.WithKind(ctx, "message")
		ctx = logctx.WithChatID(ctx, update.Message.Chat.ID)
		ctx = logctx.WithThreadID(ctx, update.Message.MessageThreadID)
		ctx = logctx.WithUserID(ctx, update.Message.From.ID)

		if cmd := commandOf(update.Message); cmd != "" {
			ctx = logctx.WithOp(ctx, "command."+cmd)
		}

	case update.CallbackQuery != nil:
		ctx = logctx.WithKind(ctx, "callback_query")
		ctx = logctx.WithUserID(ctx, update.CallbackQuery.From.ID)

		if m := update.CallbackQuery.Message; m != nil {
			if msg := m.Message(); msg != nil {
				ctx = logctx.WithChatID(ctx, msg.Chat.ID)
				ctx = logctx.WithThreadID(ctx, msg.MessageThreadID)
			}
		}

		if action, _, _ := strings.Cut(update.CallbackQuery.Data, ":"); action != "" {
			ctx = logctx.WithOp(ctx, "callback."+action)
		}

	case update.MyChatMember != nil:
		ctx = logctx.WithKind(ctx, "my_chat_member")
		ctx = logctx.WithOp(ctx, "my_chat_member")
		ctx = logctx.WithChatID(ctx, update.MyChatMember.Chat.ID)
		ctx = logctx.WithUserID(ctx, update.MyChatMember.From.ID)

	default:
		ctx = logctx.WithKind(ctx, "other")
	}

	return ctx
}

func commandOf(msg *telego.Message) string {
	text := msg.Text
	if text == "" {
		text = msg.Caption
	}

	if !strings.HasPrefix(text, "/") {
		return ""
	}

	cmd, _, _ := tu.ParseCommand(text)

	return cmd
}
