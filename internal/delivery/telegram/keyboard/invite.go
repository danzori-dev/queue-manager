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

func InviteEvents(events []domain.Event) *telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0, len(events)+1)

	for _, e := range events {
		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					fmt.Sprintf("#%d %s", e.ID, e.Title), callback.Action{
						Kind:    callback.ActionInviteEvent,
						EventID: e.ID,
					},
				),
			),
		)
	}

	rows = append(rows, cancelRow(callback.ActionInviteAbort, 0))

	return tu.InlineKeyboard(rows...)
}

func InvitePeople(c appqueue.InviteCandidates, requested int) (*telego.InlineKeyboardMarkup, Page) {
	p := paginate(len(c.Students), requested, pageSize)
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+2)

	for _, st := range c.Students[p.start:p.end] {
		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					st.Name, callback.Action{
						Kind:    callback.ActionInvitePick,
						EventID: c.Event.ID,
						UserID:  st.ID,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) callback.Action {
			return callback.Action{
				Kind:    callback.ActionInvitePage,
				EventID: c.Event.ID,
				Page:    index,
			}
		},
	)

	rows = append(rows, cancelRow(callback.ActionInviteAbort, c.Event.ID))

	return tu.InlineKeyboard(rows...), p
}

func InviteConfirm(eventID, toID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonInviteSend, callback.Action{
					Kind:    callback.ActionInviteSend,
					EventID: eventID,
					UserID:  toID,
				},
			).WithStyle(telego.ButtonStyleSuccess),
			button(
				texts.ButtonCancel, callback.Action{
					Kind:    callback.ActionInviteAbort,
					EventID: eventID,
				},
			),
		),
	)
}

func InviteCancel(requestID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonInviteCancel, callback.Action{
					Kind:    callback.ActionInviteCancel,
					EventID: requestID,
				},
			).WithStyle(telego.ButtonStyleDanger),
		),
	)
}

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
