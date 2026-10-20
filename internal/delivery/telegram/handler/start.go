package handler

import (
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

const swapStartPrefix = "swap_"

func (h *Handler) HandleStart(ctx *th.Context, msg telego.Message) error {
	if eventID, ok := swapStartPayload(msg.Text); ok {
		return h.startSwap(ctx, msg, eventID)
	}

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

func swapStartPayload(text string) (int64, bool) {
	_, _, args := tu.ParseCommand(text)
	if len(args) != 1 {
		return 0, false
	}

	raw, ok := strings.CutPrefix(args[0], swapStartPrefix)
	if !ok {
		return 0, false
	}

	eventID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}

	return eventID, true
}
