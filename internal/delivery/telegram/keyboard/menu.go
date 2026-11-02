package keyboard

import (
	"fmt"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
	"github.com/danzori-dev/queue-manager/internal/domain/student"
)

func Events(events []domain.Event, requested int) (*telego.InlineKeyboardMarkup, Page) {
	p := paginate(len(events), requested, pageSize)
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+1)

	for _, e := range events[p.start:p.end] {
		label := fmt.Sprintf("#%d %s", e.ID, e.Title)
		if e.Status == domain.StatusClosed {
			label = "🔒 " + label
		}

		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					label, callback.Action{
						Kind:       callback.ActionMenu,
						EventID:    e.ID,
						EventsPage: p.Index,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) callback.Action {
			return callback.Action{
				Kind: callback.ActionList,
				Page: index,
			}
		},
	)

	return tu.InlineKeyboard(rows...), p
}

func Menu(view appqueue.View, eventsPage int) *telego.InlineKeyboardMarkup {
	eventID := view.Event.ID

	toggle := button(
		texts.ButtonClose, callback.Action{
			Kind:       callback.ActionClose,
			EventID:    eventID,
			EventsPage: eventsPage,
		},
	).WithStyle(telego.ButtonStyleDanger)

	if view.Event.Status == domain.StatusClosed {
		toggle = button(
			texts.ButtonOpen, callback.Action{
				Kind:       callback.ActionOpen,
				EventID:    eventID,
				EventsPage: eventsPage,
			},
		).WithStyle(telego.ButtonStyleSuccess)
	}

	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			toggle, button(
				texts.ButtonRefresh, callback.Action{
					Kind:       callback.ActionMenu,
					EventID:    eventID,
					EventsPage: eventsPage,
				},
			),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonMembers, callback.Action{
					Kind:       callback.ActionMembers,
					EventID:    eventID,
					EventsPage: eventsPage,
				},
			),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonRename, callback.Action{
					Kind:       callback.ActionRename,
					EventID:    eventID,
					EventsPage: eventsPage,
				},
			),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonDelete, callback.Action{
					Kind:       callback.ActionDelete,
					EventID:    eventID,
					EventsPage: eventsPage,
				},
			).WithStyle(telego.ButtonStyleDanger),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonBack, callback.Action{
					Kind: callback.ActionList,
					Page: eventsPage,
				},
			),
		),
	)
}

func DeleteConfirm(eventID int64, eventsPage int) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonConfirmDelete, callback.Action{
					Kind:       callback.ActionDeleteConfirm,
					EventID:    eventID,
					EventsPage: eventsPage,
				},
			).WithStyle(telego.ButtonStyleDanger),
			button(
				texts.ButtonCancel, callback.Action{
					Kind:       callback.ActionMenu,
					EventID:    eventID,
					EventsPage: eventsPage,
				},
			),
		),
	)
}

func RenameCancel(eventID int64, eventsPage int) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonCancel, callback.Action{
					Kind:       callback.ActionRenameCancel,
					EventID:    eventID,
					EventsPage: eventsPage,
				},
			),
		),
	)
}

func Members(view appqueue.View, requested int, selfID int64, eventsPage int) (*telego.InlineKeyboardMarkup, Page) {
	eventID := view.Event.ID
	p := paginate(len(view.Lines), requested, pageSize)
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+3)

	for _, line := range view.Lines[p.start:p.end] {
		label := fmt.Sprintf("❌ %d. %s", line.Position, line.Name)
		if line.UserID == selfID {
			label += " (ты)"
		}

		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					label, callback.Action{
						Kind:       callback.ActionKick,
						EventID:    eventID,
						UserID:     line.UserID,
						Page:       p.Index,
						EventsPage: eventsPage,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) callback.Action {
			return callback.Action{
				Kind:       callback.ActionMembers,
				EventID:    eventID,
				Page:       index,
				EventsPage: eventsPage,
			}
		},
	)

	rows = append(
		rows,
		tu.InlineKeyboardRow(
			button(
				texts.ButtonAddParticipant, callback.Action{
					Kind:       callback.ActionParticipantCandidates,
					EventID:    eventID,
					EventsPage: eventsPage,
				},
			),
		),
		tu.InlineKeyboardRow(
			button(
				texts.ButtonBackToEvent, callback.Action{
					Kind:       callback.ActionMenu,
					EventID:    eventID,
					EventsPage: eventsPage,
				},
			),
		),
	)

	return tu.InlineKeyboard(rows...), p
}

func ParticipantCandidates(students []student.Student, eventID int64, requested, eventsPage int) (*telego.InlineKeyboardMarkup, Page) {
	p := paginate(len(students), requested, pageSize)
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+2)

	for _, st := range students[p.start:p.end] {
		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					st.Name, callback.Action{
						Kind:       callback.ActionParticipantAdd,
						EventID:    eventID,
						UserID:     st.ID,
						EventsPage: eventsPage,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) callback.Action {
			return callback.Action{
				Kind:       callback.ActionParticipantCandidates,
				EventID:    eventID,
				Page:       index,
				EventsPage: eventsPage,
			}
		},
	)

	rows = append(
		rows, tu.InlineKeyboardRow(
			button(
				texts.ButtonBackToMembers, callback.Action{
					Kind:       callback.ActionMembers,
					EventID:    eventID,
					EventsPage: eventsPage,
				},
			),
		),
	)

	return tu.InlineKeyboard(rows...), p
}
