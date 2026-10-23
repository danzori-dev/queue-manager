package slog

import (
	"context"
	"log/slog"
	"os"
	"strings"
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

func New(ctx context.Context, opts Options) *slog.Logger {
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

	if opts.Alerts != nil {
		handler = newAlertHandler(ctx, handler, opts.Alerts, secrets)
	}

	return slog.New(newContextHandler(handler)).With(slog.String("version", opts.Version))
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
