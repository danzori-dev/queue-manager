package keyboard

import (
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func Queue(view appqueue.View, botUsername string) *telego.InlineKeyboardMarkup {
	if view.Event.Status == domain.StatusClosed {
		return None()
	}

	swapURL := fmt.Sprintf("https://t.me/%s?start=%s", botUsername, callback.SwapStart(view.Event.ID))
	inviteURL := fmt.Sprintf("https://t.me/%s?start=%s", botUsername, callback.InviteStart(view.Event.ID))

	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonJoin, callback.Action{
					Kind:    callback.ActionJoin,
					EventID: view.Event.ID,
				},
			).WithStyle(telego.ButtonStyleSuccess),
			button(
				texts.ButtonLeave, callback.Action{
					Kind:    callback.ActionLeave,
					EventID: view.Event.ID,
				},
			).WithStyle(telego.ButtonStyleDanger),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonSkip, callback.Action{
					Kind:    callback.ActionSkip,
					EventID: view.Event.ID,
				},
			),
			tu.InlineKeyboardButton(texts.ButtonSwap).WithURL(swapURL),
		),
		tu.InlineKeyboardRow(tu.InlineKeyboardButton(texts.ButtonInvite).WithURL(inviteURL)),
	)
}
