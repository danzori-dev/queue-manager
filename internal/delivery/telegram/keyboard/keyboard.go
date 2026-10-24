package keyboard

import (
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
)

func None() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{InlineKeyboard: [][]telego.InlineKeyboardButton{}}
}

func Back() *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(tu.InlineKeyboardRow(button(texts.ButtonBack, callback.Action{Kind: callback.ActionList})))
}

func button(label string, action callback.Action) telego.InlineKeyboardButton {
	return tu.InlineKeyboardButton(label).WithCallbackData(action.String())
}
