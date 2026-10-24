package middleware

import (
	"context"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

type StudentLookup interface {
	Name(userID int64) (string, bool)
}

func WhiteList(students StudentLookup, bot *telego.Bot, apiTimeout time.Duration) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		if update.MyChatMember != nil || RoleFromContext(ctx).AtLeast(access.RoleAdmin) {
			return ctx.Next(update)
		}

		if _, ok := students.Name(extractUserID(update)); ok {
			return ctx.Next(update)
		}

		if update.CallbackQuery != nil {
			reqCtx, cancel := context.WithTimeout(ctx, apiTimeout)
			defer cancel()

			_ = bot.AnswerCallbackQuery(reqCtx, tu.CallbackQuery(update.CallbackQuery.ID).WithText(texts.NotWhitelisted))
		}

		return nil
	}
}
