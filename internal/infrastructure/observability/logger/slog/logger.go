package slog

import (
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
}

func New(opts Options) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{
		Level:       opts.Level,
		ReplaceAttr: redact(opts.Secrets),
	}

	var handler slog.Handler

	if opts.Format == FormatText {
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	return slog.New(newContextHandler(handler)).With(slog.String("version", opts.Version))
}

func redact(secrets []string) func([]string, slog.Attr) slog.Attr {
	pairs := make([]string, 0, len(secrets)*2)
	for _, s := range secrets {
		if s != "" {
			pairs = append(pairs, s, "***")
		}
	}

	if len(pairs) == 0 {
		return nil
	}

	r := strings.NewReplacer(pairs...)

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
