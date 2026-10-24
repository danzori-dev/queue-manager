package keyboard

import (
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
)

func Carry(fromID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonCarry, callback.Action{
					Kind:    callback.ActionCarry,
					EventID: fromID,
				},
			),
			button(
				texts.ButtonCancel, callback.Action{
					Kind:    callback.ActionCarryCancel,
					EventID: fromID,
				},
			),
		),
	)
}
