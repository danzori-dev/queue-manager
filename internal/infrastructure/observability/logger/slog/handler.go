package slog

import (
	"context"
	"log/slog"

	"github.com/danzori-dev/queue-manager/internal/application/logctx"
)

type contextHandler struct {
	next slog.Handler
}

func newContextHandler(next slog.Handler) *contextHandler {
	return &contextHandler{
		next: next,
	}
}

func (h *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *contextHandler) Handle(ctx context.Context, rec slog.Record) error {
	if attrs := logctx.Attrs(ctx); len(attrs) > 0 {
		rec = rec.Clone()
		rec.AddAttrs(attrs...)
	}

	return h.next.Handle(ctx, rec)
}

func (h *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{
		next: h.next.WithAttrs(attrs),
	}
}

func (h *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{
		next: h.next.WithGroup(name),
	}
}
