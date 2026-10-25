package general

import (
	"log/slog"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/command"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/reply"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/server/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

type Deps struct {
	Replier *reply.Replier
	Queues  QueueService
	Swaps   Starter
	Invites Starter
	Version string
	Log     *slog.Logger
}

type Handler struct {
	reply   *reply.Replier
	queues  QueueService
	swaps   Starter
	invites Starter
	version string
	log     *slog.Logger
}

func New(d Deps) *Handler {
	return &Handler{
		reply:   d.Replier,
		queues:  d.Queues,
		swaps:   d.Swaps,
		invites: d.Invites,
		version: d.Version,
		log:     d.Log,
	}
}

func (h *Handler) Start(ctx *th.Context, msg telego.Message) error {
	if payload, ok := command.StartPayload(msg.Text); ok {
		if eventID, ok := callback.ParseSwapStart(payload); ok {
			return h.swaps.Start(ctx, msg, eventID)
		}

		if eventID, ok := callback.ParseInviteStart(payload); ok {
			return h.invites.Start(ctx, msg, eventID)
		}
	}

	return h.reply.SendHTML(ctx, msg, texts.Welcome)
}

func (h *Handler) Help(ctx *th.Context, msg telego.Message) error {
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

	return h.reply.SendHTML(ctx, msg, texts.Help(sections...))
}

func (h *Handler) Status(ctx *th.Context, msg telego.Message) error {
	if ok, err := h.reply.Allowed(ctx, msg, access.RoleOwner); !ok {
		return err
	}

	start := time.Now()
	ids, err := h.queues.OpenEventIDs(ctx)

	return h.reply.SendHTML(
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

func (h *Handler) MyChatMember(ctx *th.Context, update telego.ChatMemberUpdated) error {
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
