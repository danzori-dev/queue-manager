package keyboard

import (
	"fmt"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

func CarrySources(events []domain.Event, requested int, selectedFromID int64) (*telego.InlineKeyboardMarkup, Page) {
	p := paginate(len(events), requested, pageSize)
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+2)

	for _, event := range events[p.start:p.end] {
		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					event.Title, callback.Action{
						Kind:    callback.ActionCarrySource,
						EventID: event.ID,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) callback.Action {
			return callback.Action{
				Kind: callback.ActionCarrySourcesPage,
				Page: index,
			}
		},
	)

	rows = append(rows, CarryCancel(selectedFromID).InlineKeyboard...)

	return tu.InlineKeyboard(rows...), p
}

func CarryPlaces(view appqueue.View, requested int) (*telego.InlineKeyboardMarkup, Page) {
	places := make([][]appqueue.Line, 0)

	for _, line := range view.Lines {
		last := len(places) - 1

		if last < 0 || places[last][0].Position != line.Position {
			places = append(places, []appqueue.Line{line})

			continue
		}

		places[last] = append(places[last], line)
	}

	p := paginate(len(places), requested, pageSize)
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+2)

	for _, place := range places[p.start:p.end] {
		names := make([]string, len(place))

		for i, line := range place {
			names[i] = line.Name
		}

		label := fmt.Sprintf("%d. %s", place[0].Position, strings.Join(names, ", "))

		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					label, callback.Action{
						Kind:     callback.ActionCarryPlace,
						EventID:  view.Event.ID,
						Position: place[0].Position,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) callback.Action {
			return callback.Action{
				Kind:    callback.ActionCarryPlacesPage,
				EventID: view.Event.ID,
				Page:    index,
			}
		},
	)

	rows = append(rows, CarryBackToSources(view.Event.ID).InlineKeyboard...)

	return tu.InlineKeyboard(rows...), p
}

func CarryTargets(events []domain.Event, fromID int64, fromPos int, requested int) (*telego.InlineKeyboardMarkup, Page) {
	p := paginate(len(events), requested, pageSize)
	rows := make([][]telego.InlineKeyboardButton, 0, p.end-p.start+2)

	for _, event := range events[p.start:p.end] {
		rows = append(
			rows, tu.InlineKeyboardRow(
				button(
					fmt.Sprintf("#%d %s", event.ID, event.Title), callback.Action{
						Kind:    callback.ActionCarryTarget,
						EventID: event.ID,
					},
				),
			),
		)
	}

	rows = appendPageRow(
		rows, p, func(index int) callback.Action {
			return callback.Action{
				Kind:     callback.ActionCarryTargetsPage,
				EventID:  fromID,
				Position: fromPos,
				Page:     index,
			}
		},
	)

	rows = append(rows, CarryBackToPlaces(fromID, PageOf(fromPos-1)).InlineKeyboard...)

	return tu.InlineKeyboard(rows...), p
}

func CarryBackToSources(fromID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonBack, callback.Action{
					Kind: callback.ActionCarrySourcesPage,
					Page: 0,
				},
			),
			button(
				texts.ButtonCancel, callback.Action{
					Kind:    callback.ActionCarryCancel,
					EventID: fromID,
				},
			).WithStyle(telego.ButtonStyleDanger),
		),
	)
}

func CarryBackToPlaces(fromID int64, page int) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonBack, callback.Action{
					Kind:    callback.ActionCarryPlacesPage,
					EventID: fromID,
					Page:    page,
				},
			),
			button(
				texts.ButtonCancel, callback.Action{
					Kind:    callback.ActionCarryCancel,
					EventID: fromID,
				},
			).WithStyle(telego.ButtonStyleDanger),
		),
	)
}

func CarryCancel(fromID int64) *telego.InlineKeyboardMarkup {
	return tu.InlineKeyboard(
		tu.InlineKeyboardRow(
			button(
				texts.ButtonCancel, callback.Action{
					Kind:    callback.ActionCarryCancel,
					EventID: fromID,
				},
			).WithStyle(telego.ButtonStyleDanger),
		),
	)
}

func Carry(plan appqueue.CarryPlan, targetPage, requested int) (*telego.InlineKeyboardMarkup, Page) {
	var (
		fromPos    int
		placeCount int
	)

	if len(plan.Lines) > 0 {
		fromPos = plan.Lines[0].Position
		placeCount = plan.Lines[len(plan.Lines)-1].Position - fromPos + 1
	}

	p := paginate(placeCount, requested, pageSize)
	rows := make([][]telego.InlineKeyboardButton, 0, 3)

	rows = appendPageRow(
		rows, p, func(index int) callback.Action {
			return callback.Action{
				Kind:    callback.ActionCarryPlanPage,
				EventID: plan.To.ID,
				Page:    index,
			}
		},
	)

	rows = append(
		rows, tu.InlineKeyboardRow(
			button(
				texts.ButtonCarry, callback.Action{
					Kind:    callback.ActionCarry,
					EventID: plan.From.ID,
				},
			).WithStyle(telego.ButtonStyleSuccess),
			button(
				texts.ButtonCancel, callback.Action{
					Kind:    callback.ActionCarryCancel,
					EventID: plan.From.ID,
				},
			).WithStyle(telego.ButtonStyleDanger),
		),
	)

	rows = append(
		rows, tu.InlineKeyboardRow(
			button(
				texts.ButtonBack, callback.Action{
					Kind:     callback.ActionCarryTargetsPage,
					EventID:  plan.From.ID,
					Position: fromPos,
					Page:     targetPage,
				},
			),
		),
	)

	return tu.InlineKeyboard(rows...), p
}
