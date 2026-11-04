package slog

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

type Options struct {
	Level   slog.Level
	Format  Format
	Version string
	Secrets []string
	Alerts  AlertSender
}

type Logger struct {
	*slog.Logger

	alerts *alertHandler
	sender AlertSender
}

func New(ctx context.Context, opts Options) *Logger {
	secrets := secretsReplacer(opts.Secrets)

	handlerOpts := &slog.HandlerOptions{
		Level:       opts.Level,
		ReplaceAttr: redact(secrets),
	}

	var handler slog.Handler

	if opts.Format == FormatText {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	var alerts *alertHandler

	if opts.Alerts != nil {
		alerts = newAlertHandler(ctx, handler, opts.Alerts, secrets)
		handler = alerts
	}

	log := slog.New(newContextHandler(handler)).
		With(slog.String("version", opts.Version))

	return &Logger{
		Logger: log,
		alerts: alerts,
		sender: opts.Alerts,
	}
}

func (l *Logger) Shutdown(ctx context.Context, cause error) error {
	if l.alerts != nil {
		l.alerts.queue.stop()
	}

	if cause != nil {
		l.ErrorContext(
			ctx, "application stopped",
			slog.Any("err", cause),
		)
	}

	if l.alerts == nil {
		return nil
	}

	if err := l.alerts.queue.wait(ctx); err != nil {
		return fmt.Errorf("wait for owner alerts: %w", err)
	}

	if cause == nil {
		return nil
	}

	rec := slog.NewRecord(time.Now(), slog.LevelError, "application stopped", 0)
	rec.AddAttrs(slog.Any("err", cause))

	if err := l.sender.Send(ctx, l.alerts.format(rec)); err != nil {
		return fmt.Errorf("send final owner alert: %w", err)
	}

	return nil
}

func secretsReplacer(secrets []string) *strings.Replacer {
	pairs := make([]string, 0, len(secrets)*2)

	for _, s := range secrets {
		if s != "" {
			pairs = append(pairs, s, "***")
		}
	}

	return strings.NewReplacer(pairs...)
}

func redact(r *strings.Replacer) func([]string, slog.Attr) slog.Attr {
	return func(_ []string, a slog.Attr) slog.Attr {
		switch a.Value.Kind() {
		case slog.KindString:
			return slog.String(a.Key, r.Replace(a.Value.String()))

		case slog.KindAny:
			if err, ok := a.Value.Any().(error); ok {
				return slog.String(a.Key, r.Replace(err.Error()))
			}

			return a

		default:
			return a
		}
	}
}
