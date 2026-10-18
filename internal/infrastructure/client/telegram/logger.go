package telegram

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

type logger struct {
	log *slog.Logger
}

func (l logger) Debugf(string, ...any) {}

func (l logger) Errorf(format string, args ...any) {
	for _, arg := range args {
		if err, ok := arg.(error); ok && errors.Is(err, context.Canceled) {
			return
		}
	}

	msg := fmt.Sprintf(format, args...)

	if strings.HasPrefix(format, "Execution error") {
		l.log.Debug(msg)

		return
	}

	l.log.Error(msg)
}
