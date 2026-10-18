package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
)

func Recover(bot *telego.Bot, apiTimeout time.Duration, log *slog.Logger) th.Handler {
	return func(ctx *th.Context, update telego.Update) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.ErrorContext(
					ctx.Context(), "panic recovered in telegram handler",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)

				if update.CallbackQuery != nil {
					cbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), apiTimeout)
					defer cancel()

					_ = bot.AnswerCallbackQuery(
						cbCtx,
						tu.CallbackQuery(update.CallbackQuery.ID).WithText(texts.InternalError),
					)
				}

				err = fmt.Errorf("panic recovered: %v", r)
			}
		}()

		return ctx.Next(update)
	}
}
