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

func Events(events []domain.Event) *telego.InlineKeyboardMarkup {
	rows := make([][]telego.InlineKeyboardButton, 0, len(events))

	for _, e := range events {
		label := fmt.Sprintf("#%d %s", e.ID, e.Title)
		if e.Status == domain.StatusClosed {
			label = "🔒 " + label
		}

		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					label, callback.Action{
						Kind:    callback.ActionMenu,
						EventID: e.ID,
					},
				),
			),
		)
	}

	return tu.InlineKeyboard(rows...)
}

func Menu(view appqueue.View) *telego.InlineKeyboardMarkup {
	eventID := view.Event.ID

	toggle := button(
		texts.ButtonClose, callback.Action{
			Kind:    callback.ActionClose,
			EventID: eventID,
		},
	).WithStyle(telego.ButtonStyleDanger)

	if view.Event.Status == domain.StatusClosed {
		toggle = button(
			texts.ButtonOpen, callback.Action{
				Kind:    callback.ActionOpen,
				EventID: eventID,
			},
		).WithStyle(telego.ButtonStyleSuccess)
	}

	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			toggle, button(
				texts.ButtonRefresh, callback.Action{
					Kind:    callback.ActionMenu,
					EventID: eventID,
				},
			),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonMembers, callback.Action{
					Kind:    callback.ActionMembers,
					EventID: eventID,
				},
			),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonRename, callback.Action{
					Kind:    callback.ActionRename,
					EventID: eventID,
				},
			),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonDelete, callback.Action{
					Kind:    callback.ActionDelete,
					EventID: eventID,
				},
			).WithStyle(telego.ButtonStyleDanger),
		),
		tu.InlineKeyboardRow(button(texts.ButtonBack, callback.Action{Kind: callback.ActionList})),
	)
}

func DeleteConfirm(eventID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonConfirmDelete, callback.Action{
					Kind:    callback.ActionDeleteConfirm,
					EventID: eventID,
				},
			).WithStyle(telego.ButtonStyleDanger),
			button(
				texts.ButtonCancel, callback.Action{
					Kind:    callback.ActionMenu,
					EventID: eventID,
				},
			),
		),
	)
}

func RenameCancel(eventID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonCancel, callback.Action{
					Kind:    callback.ActionRenameCancel,
					EventID: eventID,
				},
			),
		),
	)
}

func Members(view appqueue.View, requested int, selfID int64) (*telego.InlineKeyboardMarkup, Page) {
	eventID := view.Event.ID
	p := paginate(len(view.Lines), requested)
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+2)

	for _, line := range view.Lines[p.start:p.end] {
		label := fmt.Sprintf("❌ %d. %s", line.Position, line.Name)
		if line.UserID == selfID {
			label += " (ты)"
		}

		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					label, callback.Action{
						Kind:    callback.ActionKick,
						EventID: eventID,
						UserID:  line.UserID,
						Page:    p.Index,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) callback.Action {
			return callback.Action{
				Kind:    callback.ActionMembers,
				EventID: eventID,
				Page:    index,
			}
		},
	)

	rows = append(
		rows, tu.InlineKeyboardRow(
			button(
				texts.ButtonBackToEvent, callback.Action{
					Kind:    callback.ActionMenu,
					EventID: eventID,
				},
			),
		),
	)

	return tu.InlineKeyboard(rows...), p
}
