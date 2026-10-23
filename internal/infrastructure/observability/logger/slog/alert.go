package slog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

const (
	alertQueueSize = 32
	stackKey       = "stack"
)

type AlertSender interface {
	Send(ctx context.Context, text string) error
}

type alertHandler struct {
	next    slog.Handler
	alerts  chan<- string
	secrets *strings.Replacer
}

func newAlertHandler(ctx context.Context, next slog.Handler, sender AlertSender, secrets *strings.Replacer) *alertHandler {
	alerts := make(chan string, alertQueueSize)
	ctx = context.WithoutCancel(ctx)

	go func() {
		for text := range alerts {
			_ = sender.Send(ctx, text)
		}
	}()

	return &alertHandler{
		next:    next,
		alerts:  alerts,
		secrets: secrets,
	}
}

func (h *alertHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *alertHandler) Handle(ctx context.Context, rec slog.Record) error {
	err := h.next.Handle(ctx, rec)

	if rec.Level >= slog.LevelWarn {
		h.alert(rec)
	}

	return err
}

func (h *alertHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &alertHandler{
		next:    h.next.WithAttrs(attrs),
		alerts:  h.alerts,
		secrets: h.secrets,
	}
}

func (h *alertHandler) WithGroup(name string) slog.Handler {
	return &alertHandler{
		next:    h.next.WithGroup(name),
		alerts:  h.alerts,
		secrets: h.secrets,
	}
}

func (h *alertHandler) alert(rec slog.Record) {
	select {
	case h.alerts <- h.format(rec):
	default:
	}
}

func (h *alertHandler) format(rec slog.Record) string {
	icon := "🟡"

	if rec.Level >= slog.LevelError {
		icon = "🔴"
	}

	var b strings.Builder

	fmt.Fprintf(&b, "%s %s: %s", icon, rec.Level, rec.Message)

	rec.Attrs(
		func(a slog.Attr) bool {
			if a.Key != stackKey {
				fmt.Fprintf(&b, "\n%s: %s", a.Key, a.Value)
			}

			return true
		},
	)

	return h.secrets.Replace(b.String())
}
