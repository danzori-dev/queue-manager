package middleware

import (
	"context"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"

	"github.com/danzori-dev/queue-manager/internal/domain/access"
)

type roleCtxKey struct{}

type RoleResolver interface {
	RoleOf(userID int64) access.Role
}

func Roles(resolver RoleResolver) th.Handler {
	return func(ctx *th.Context, update telego.Update) error {
		userID := extractUserID(update)
		if userID == 0 {
			return ctx.Next(update)
		}

		stdCtx := context.WithValue(ctx, roleCtxKey{}, resolver.RoleOf(userID))

		return ctx.WithContext(stdCtx).Next(update)
	}
}

func RoleFromContext(ctx context.Context) access.Role {
	if role, ok := ctx.Value(roleCtxKey{}).(access.Role); ok {
		return role
	}

	return access.RoleOutsider
}

func extractUserID(update telego.Update) int64 {
	switch {
	case update.Message != nil:
		return update.Message.From.ID

	case update.CallbackQuery != nil:
		return update.CallbackQuery.From.ID

	case update.MyChatMember != nil:
		return update.MyChatMember.From.ID

	default:
		return 0
	}
}
