package middleware

import (
	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

type StudentLookup interface {
	Name(userID int64) (string, bool)
}

func WhiteList(students StudentLookup) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		if update.MyChatMember != nil {
			return ctx.Next(update)
		}

		if RoleFromContext(ctx).AtLeast(access.RoleAdmin) {
			return ctx.Next(update)
		}

		if _, ok := students.Name(extractUserID(update)); ok {
			return ctx.Next(update)
		}

		return nil
	}
}
