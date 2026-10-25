package middleware

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
	tu "github.com/mymmrac/telego/telegoutil"
	"golang.org/x/time/rate"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

type RateLimitConfig struct {
	Burst    int
	Interval time.Duration
}

func RateLimit(student, admin RateLimitConfig, bot *telego.Bot, apiTimeout time.Duration, log *slog.Logger) th.Handler {
	var (
		mu       sync.Mutex
		limiters = map[access.Role]map[int64]*rate.Limiter{
			access.RoleStudent: {},
			access.RoleAdmin:   {},
		}
	)

	allow := func(userID int64, role access.Role) bool {
		cfg, byUser := student, limiters[access.RoleStudent]

		if role.AtLeast(access.RoleAdmin) {
			cfg, byUser = admin, limiters[access.RoleAdmin]
		}

		mu.Lock()
		defer mu.Unlock()

		limiter, ok := byUser[userID]
		if !ok {
			limiter = rate.NewLimiter(rate.Every(cfg.Interval), cfg.Burst)
			byUser[userID] = limiter
		}

		return limiter.Allow()
	}

	return func(ctx *th.Context, update telego.Update) error {
		userID := extractUserID(update)
		role := RoleFromContext(ctx)

		if userID == 0 || update.MyChatMember != nil || role.AtLeast(access.RoleOwner) {
			return ctx.Next(update)
		}

		if allow(userID, role) {
			return ctx.Next(update)
		}

		log.InfoContext(ctx, "rate limited")

		if update.CallbackQuery != nil {
			reqCtx, cancel := context.WithTimeout(ctx, apiTimeout)
			defer cancel()

			_ = bot.AnswerCallbackQuery(reqCtx, tu.CallbackQuery(update.CallbackQuery.ID).WithText(texts.TooManyRequests))
		}

		return nil
	}
}
