package handler

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

func (h *Handler) HandleStart(ctx *th.Context, msg telego.Message) error {
	return h.sendHTML(ctx, msg, texts.Welcome)
}

func (h *Handler) HandleHelp(ctx *th.Context, msg telego.Message) error {
	sections := []string{texts.HelpCommon}

	if msg.Chat.Type == telego.ChatTypePrivate {
		role := middleware.RoleFromContext(ctx)

		if role.AtLeast(access.RoleAdmin) {
			sections = append(sections, texts.HelpAdmin)
		}

		if role.AtLeast(access.RoleOwner) {
			sections = append(sections, texts.HelpOwner)
		}
	}

	return h.sendHTML(ctx, msg, texts.Help(sections...))
}
