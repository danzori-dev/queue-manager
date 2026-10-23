package logctx

import (
	"context"
	"log/slog"
)

type ctxKey struct{}

func With(ctx context.Context, attrs ...slog.Attr) context.Context {
	if len(attrs) == 0 {
		return ctx
	}

	existing := Attrs(ctx)

	merged := make([]slog.Attr, 0, len(existing)+len(attrs))
	merged = append(merged, existing...)
	merged = append(merged, attrs...)

	return context.WithValue(ctx, ctxKey{}, merged)
}

func Attrs(ctx context.Context) []slog.Attr {
	attrs, ok := ctx.Value(ctxKey{}).([]slog.Attr)
	if !ok {
		return nil
	}

	return attrs
}

func WithOp(ctx context.Context, op string) context.Context {
	return With(ctx, slog.String("op", op))
}

func WithKind(ctx context.Context, kind string) context.Context {
	return With(ctx, slog.String("kind", kind))
}

func WithUpdateID(ctx context.Context, updateID int) context.Context {
	return With(ctx, slog.Int("update_id", updateID))
}

func WithChatID(ctx context.Context, chatID int64) context.Context {
	return With(ctx, slog.Int64("chat_id", chatID))
}

func WithThreadID(ctx context.Context, threadID int) context.Context {
	if threadID == 0 {
		return ctx
	}

	return With(ctx, slog.Int("thread_id", threadID))
}

func WithUserID(ctx context.Context, userID int64) context.Context {
	return With(ctx, slog.Int64("user_id", userID))
}

func WithEventID(ctx context.Context, eventID int64) context.Context {
	return With(ctx, slog.Int64("event_id", eventID))
}
