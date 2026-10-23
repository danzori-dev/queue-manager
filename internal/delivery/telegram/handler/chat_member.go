package handler

import (
	"log/slog"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

func (h *Handler) HandleMyChatMember(ctx *th.Context, update telego.ChatMemberUpdated) error {
	if update.Chat.Type == telego.ChatTypePrivate {
		return nil
	}

	status := update.NewChatMember.MemberStatus()
	if status != telego.MemberStatusLeft && status != telego.MemberStatusBanned {
		return nil
	}

	h.log.WarnContext(
		ctx, "bot removed from chat",
		slog.String("chat_title", update.Chat.Title),
	)

	return nil
}
