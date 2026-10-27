package keyboard

import (
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
)

type ActivityRow struct {
	UserID int64
	Name   string
}

func ActivityPeople(people []ActivityRow, requested int) (*telego.InlineKeyboardMarkup, Page) {
	p := paginate(len(people), requested, pageSize)
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+1)

	for _, person := range people[p.start:p.end] {
		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					person.Name, callback.Action{
						Kind:   callback.ActionActivityLog,
						UserID: person.UserID,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) callback.Action {
			return callback.Action{
				Kind: callback.ActionActivityList,
				Page: index,
			}
		},
	)

	return tu.InlineKeyboard(rows...), p
}

func ActivityEvents(userID int64, listPage, total, requested int) (*telego.InlineKeyboardMarkup, Page) {
	p := paginate(total, requested, eventsPageSize)

	rows := appendPageRow(
		nil, p, func(index int) callback.Action {
			return callback.Action{
				Kind:   callback.ActionActivityLog,
				UserID: userID,
				Page:   index,
			}
		},
	)

	rows = append(
		rows, tu.InlineKeyboardRow(
			button(
				texts.ButtonActivityToList, callback.Action{
					Kind: callback.ActionActivityList,
					Page: listPage,
				},
			),
		),
	)

	return tu.InlineKeyboard(rows...), p
}
