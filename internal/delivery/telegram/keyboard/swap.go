package keyboard

import (
	"fmt"
	"slices"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func SwapEvents(events []domain.Event) *telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0, len(events))

	for _, e := range events {
		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					fmt.Sprintf("#%d %s", e.ID, e.Title), callback.Action{
						Kind:    callback.ActionSwapEvent,
						EventID: e.ID,
					},
				),
			),
		)
	}

	return tu.InlineKeyboard(rows...)
}

func SwapPeople(view appqueue.View, selfID int64, requested int) (*telego.InlineKeyboardMarkup, Page) {
	others := slices.DeleteFunc(
		slices.Clone(view.Lines), func(l appqueue.Line) bool {
			return l.UserID == selfID
		},
	)

	p := paginate(len(others), requested)
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+1)

	for _, line := range others[p.start:p.end] {
		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					fmt.Sprintf("%d. %s", line.Position, line.Name), callback.Action{
						Kind:    callback.ActionSwapPick,
						EventID: view.Event.ID,
						UserID:  line.UserID,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) callback.Action {
			return callback.Action{
				Kind:    callback.ActionSwapPage,
				EventID: view.Event.ID,
				Page:    index,
			}
		},
	)

	return tu.InlineKeyboard(rows...), p
}

func SwapConfirm(eventID, toID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonSwapSend, callback.Action{
					Kind:    callback.ActionSwapSend,
					EventID: eventID,
					UserID:  toID,
				},
			).WithStyle(telego.ButtonStyleSuccess),
			button(
				texts.ButtonCancel, callback.Action{
					Kind:    callback.ActionSwapAbort,
					EventID: eventID,
				},
			),
		),
	)
}

func SwapCancel(requestID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonSwapCancel, callback.Action{
					Kind:    callback.ActionSwapCancel,
					EventID: requestID,
				},
			).WithStyle(telego.ButtonStyleDanger),
		),
	)
}

func SwapAnswer(requestID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonSwapAccept, callback.Action{
					Kind:    callback.ActionSwapAccept,
					EventID: requestID,
				},
			).WithStyle(telego.ButtonStyleSuccess),
			button(
				texts.ButtonSwapDecline, callback.Action{
					Kind:    callback.ActionSwapDecline,
					EventID: requestID,
				},
			).WithStyle(telego.ButtonStyleDanger),
		),
	)
}
