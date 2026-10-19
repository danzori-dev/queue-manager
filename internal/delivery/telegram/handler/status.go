package handler

import (
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

func (h *Handler) HandleStatus(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	start := time.Now()
	ids, err := h.queue.OpenEventIDs(ctx)

	return h.sendHTML(
		ctx, msg, texts.Status(
			texts.BotStatus{
				Version:    h.version,
				DBLatency:  time.Since(start),
				DBErr:      err,
				OpenEvents: len(ids),
			},
		),
	)
}
