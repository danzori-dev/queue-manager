package keyboard

import (
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
)

func InviteAnswer(requestID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonInviteAccept, callback.Action{
					Kind:    callback.ActionInviteAccept,
					EventID: requestID,
				},
			).WithStyle(telego.ButtonStyleSuccess),
			button(
				texts.ButtonInviteDecline, callback.Action{
					Kind:    callback.ActionInviteDecline,
					EventID: requestID,
				},
			).WithStyle(telego.ButtonStyleDanger),
		),
	)
}
