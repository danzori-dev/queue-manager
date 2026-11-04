package slog

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

const (
	alertQueueSize = 32
	stackKey       = "stack"
)

type AlertSender interface {
	Send(ctx context.Context, text string) error
}

type alertQueue struct {
	mu     sync.Mutex
	closed bool

	alerts chan string
	done   chan struct{}
	cancel context.CancelFunc
}

func newAlertQueue(ctx context.Context, sender AlertSender) *alertQueue {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))

	q := &alertQueue{
		alerts: make(chan string, alertQueueSize),
		done:   make(chan struct{}),
		cancel: cancel,
	}

	go q.run(ctx, sender)

	return q
}

func (q *alertQueue) run(ctx context.Context, sender AlertSender) {
	defer close(q.done)

	for text := range q.alerts {
		if ctx.Err() != nil {
			return
		}

		_ = sender.Send(ctx, text)
	}
}

func (q *alertQueue) enqueue(text string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return
	}

	select {
	case q.alerts <- text:
	default:
	}
}

func (q *alertQueue) stop() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return
	}

	q.closed = true
	close(q.alerts)
}

func (q *alertQueue) wait(ctx context.Context) error {
	defer q.cancel()

	select {
	case <-q.done:
		return nil

	case <-ctx.Done():
		return ctx.Err()
	}
}

type alertHandler struct {
	next    slog.Handler
	queue   *alertQueue
	secrets *strings.Replacer
}

func newAlertHandler(ctx context.Context, next slog.Handler, sender AlertSender, secrets *strings.Replacer) *alertHandler {
	return &alertHandler{
		next:    next,
		queue:   newAlertQueue(ctx, sender),
		secrets: secrets,
	}
}

func (h *alertHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *alertHandler) Handle(ctx context.Context, rec slog.Record) error {
	err := h.next.Handle(ctx, rec)

	if rec.Level >= slog.LevelWarn {
		h.queue.enqueue(h.format(rec))
	}

	return err
}

func (h *alertHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &alertHandler{
		next:    h.next.WithAttrs(attrs),
		queue:   h.queue,
		secrets: h.secrets,
	}
}

func (h *alertHandler) WithGroup(name string) slog.Handler {
	return &alertHandler{
		next:    h.next.WithGroup(name),
		queue:   h.queue,
		secrets: h.secrets,
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
