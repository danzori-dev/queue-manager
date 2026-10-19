package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/middleware"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

type RouteParams struct {
	Bot          *telego.Bot
	Handler      *handler.Handler
	RoleResolver middleware.RoleResolver
	Students     middleware.StudentLookup
	APITimeout   time.Duration
	Log          *slog.Logger
}

func RegisterRoutes(p RouteParams) Register {
	return func(bh *th.BotHandler) {
		bh.Use(
			middleware.Logging(p.Log),
			middleware.Recover(p.Bot, p.APITimeout, p.Log),
			middleware.Roles(p.RoleResolver),
			middleware.WhiteList(p.Students),
		)

		bh.HandleMessage(p.Handler.HandleHelp, th.CommandEqual("help"))

		bh.HandleMyChatMember(p.Handler.HandleMyChatMember)

		bh.HandleCallbackQuery(
			p.Handler.HandleQueueCallback, th.AnyCallbackQueryWithMessage(),
			actions(domain.ActionJoin, domain.ActionLeave),
		)
		bh.HandleCallbackQuery(
			p.Handler.HandleMenuCallback, th.AnyCallbackQueryWithMessage(),
			actions(
				domain.ActionList, domain.ActionMenu, domain.ActionClose, domain.ActionOpen,
				domain.ActionDelete, domain.ActionDeleteConfirm, domain.ActionKick,
			),
		)

		group := bh.Group(groupChat)

		group.HandleMessage(p.Handler.HandleSetTopic, th.CommandEqual("settopic"))

		private := bh.Group(privateChat)

		private.HandleMessage(p.Handler.HandleStart, th.CommandEqual("start"))
		private.HandleMessage(p.Handler.HandleNew, th.CommandEqual("new"))
		private.HandleMessage(p.Handler.HandleEvents, th.CommandEqual("events"))

		private.HandleMessage(p.Handler.HandleAddAdmin, th.CommandEqual("addadmin"))
		private.HandleMessage(p.Handler.HandleRmAdmin, th.CommandEqual("rmadmin"))
		private.HandleMessage(p.Handler.HandleAdmins, th.CommandEqual("admins"))

		private.HandleMessage(p.Handler.HandleStudents, th.CommandEqual("students"))
		private.HandleMessage(p.Handler.HandleImportStudents, th.CaptionCommandEqual("import_students"))
		private.HandleMessage(p.Handler.HandleImportStudents, th.CommandEqual("import_students"))

		private.HandleMessage(p.Handler.HandleStatus, th.CommandEqual("status"))
	}
}

func privateChat(_ context.Context, update telego.Update) bool {
	return update.Message != nil && update.Message.Chat.Type == telego.ChatTypePrivate
}

func groupChat(_ context.Context, update telego.Update) bool {
	return update.Message != nil && update.Message.Chat.Type == telego.ChatTypeSupergroup
}

func actions(kinds ...domain.ActionKind) th.Predicate {
	predicates := make([]th.Predicate, 0, len(kinds))

	for _, k := range kinds {
		predicates = append(predicates, th.CallbackDataPrefix(string(k)+":"))
	}

	return th.Or(predicates...)
}
