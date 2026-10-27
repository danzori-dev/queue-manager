package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/server/middleware"
)

type RouteParams struct {
	Bot          *telego.Bot
	Handlers     handler.Handlers
	RoleResolver middleware.RoleResolver
	Students     middleware.StudentLookup
	RateLimit    RateLimit
	APITimeout   time.Duration
	Log          *slog.Logger
}

type RateLimit struct {
	Student middleware.RateLimitConfig
	Admin   middleware.RateLimitConfig
}

func RegisterRoutes(p RouteParams) Register {
	return func(bh *th.BotHandler) {
		bh.Use(
			middleware.Logging(p.Log, p.Students),
			middleware.Recover(p.Bot, p.APITimeout, p.Log),
			middleware.Roles(p.RoleResolver, p.Students),
			middleware.WhiteList(p.Bot, p.APITimeout),
			middleware.RateLimit(p.RateLimit.Student, p.RateLimit.Admin, p.Bot, p.APITimeout, p.Log),
		)

		private := bh.Group(privateChat)
		group := bh.Group(groupChat)

		bh.HandleMessage(p.Handlers.General.Help, th.CommandEqual("help"))
		private.HandleMessage(p.Handlers.General.Start, th.CommandEqual("start"))
		private.HandleMessage(p.Handlers.General.Status, th.CommandEqual("status"))
		bh.HandleMyChatMember(p.Handlers.General.MyChatMember)

		group.HandleMessage(p.Handlers.Queue.SetTopic, th.CommandEqual("settopic"))
		private.HandleMessage(p.Handlers.Queue.Create, th.CommandEqual("new"))
		bh.HandleCallbackQuery(
			p.Handlers.Queue.Callback, th.AnyCallbackQueryWithMessage(),
			actions(callback.ActionJoin, callback.ActionLeave, callback.ActionSkip),
		)

		private.HandleMessage(p.Handlers.Menu.Events, th.CommandEqual("events"))
		private.HandleMessage(p.Handlers.Menu.Rename, th.CommandEqual("rename"))
		bh.HandleCallbackQuery(
			p.Handlers.Menu.Callback, th.AnyCallbackQueryWithMessage(),
			actions(
				callback.ActionList, callback.ActionMenu, callback.ActionClose, callback.ActionOpen,
				callback.ActionRename, callback.ActionRenameCancel, callback.ActionDelete, callback.ActionDeleteConfirm,
				callback.ActionKick, callback.ActionMembers, callback.ActionCarry, callback.ActionCarryCancel,
			),
		)
		private.HandleMessage(p.Handlers.Menu.Text, th.AnyMessageWithText(), th.Not(th.AnyCommand()))

		private.HandleMessage(p.Handlers.Carry.Carry, th.CommandEqual("carry"))

		private.HandleMessage(p.Handlers.Swap.Swap, th.CommandEqual("swap"))
		bh.HandleCallbackQuery(
			p.Handlers.Swap.Callback, th.AnyCallbackQueryWithMessage(),
			actions(
				callback.ActionSwapEvent, callback.ActionSwapPick, callback.ActionSwapPage, callback.ActionSwapSend,
				callback.ActionSwapAbort, callback.ActionSwapAccept, callback.ActionSwapDecline, callback.ActionSwapCancel,
			),
		)

		private.HandleMessage(p.Handlers.Invite.Invite, th.CommandEqual("invite"))
		bh.HandleCallbackQuery(
			p.Handlers.Invite.Callback, th.AnyCallbackQueryWithMessage(),
			actions(
				callback.ActionInviteEvent, callback.ActionInvitePick, callback.ActionInvitePage, callback.ActionInviteSend,
				callback.ActionInviteAbort, callback.ActionInviteAccept, callback.ActionInviteDecline, callback.ActionInviteCancel,
			),
		)

		private.HandleMessage(p.Handlers.Activity.Activity, th.CommandEqual("activity"))
		bh.HandleCallbackQuery(
			p.Handlers.Activity.Callback, th.AnyCallbackQueryWithMessage(),
			actions(callback.ActionActivityList, callback.ActionActivityLog),
		)

		private.HandleMessage(p.Handlers.Admin.Add, th.CommandEqual("addadmin"))
		private.HandleMessage(p.Handlers.Admin.Remove, th.CommandEqual("rmadmin"))
		private.HandleMessage(p.Handlers.Admin.List, th.CommandEqual("admins"))

		private.HandleMessage(p.Handlers.Student.List, th.CommandEqual("students"))
		private.HandleMessage(p.Handlers.Student.Import, th.CaptionCommandEqual("import_students"))
		private.HandleMessage(p.Handlers.Student.Import, th.CommandEqual("import_students"))
	}
}

func privateChat(_ context.Context, update telego.Update) bool {
	return update.Message != nil && update.Message.Chat.Type == telego.ChatTypePrivate
}

func groupChat(_ context.Context, update telego.Update) bool {
	return update.Message != nil && update.Message.Chat.Type == telego.ChatTypeSupergroup
}

func actions(kinds ...callback.ActionKind) th.Predicate {
	predicates := make([]th.Predicate, 0, len(kinds))

	for _, k := range kinds {
		predicates = append(predicates, th.CallbackDataPrefix(string(k)+":"))
	}

	return th.Or(predicates...)
}
