package slog

import (
	"context"
	"fmt"
	"log/slog"
)

type AlertSender interface {
	Send(ctx context.Context, text string) error
}

type alertHandler struct {
	next   slog.Handler
	sender AlertSender
}

func (h *alertHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *alertHandler) Handle(ctx context.Context, rec slog.Record) error {
	err := h.next.Handle(ctx, rec)

	if rec.Level >= slog.LevelWarn {
		h.alert(ctx, rec)
	}

	return err
}

func (h *alertHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &alertHandler{
		next:   h.next.WithAttrs(attrs),
		sender: h.sender,
	}
}

func (h *alertHandler) WithGroup(name string) slog.Handler {
	return &alertHandler{
		next:   h.next.WithGroup(name),
		sender: h.sender,
	}
}

func (h *alertHandler) alert(ctx context.Context, rec slog.Record) {
	icon := "🟡"

	if rec.Level >= slog.LevelError {
		icon = "🔴"
	}

	_ = h.sender.Send(context.WithoutCancel(ctx), fmt.Sprintf("%s %s: %s", icon, rec.Level, rec.Message))
}
