package keyboard

import (
	"github.com/mymmrac/telego"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
)

const pageSize = 8

type Page struct {
	Index int
	Count int
	start int
	end   int
}

func paginate(total, requested int) Page {
	count := max(1, (total+pageSize-1)/pageSize)
	index := min(max(requested, 0), count-1)
	start := index * pageSize

	return Page{
		Index: index,
		Count: count,
		start: start,
		end:   min(start+pageSize, total),
	}
}

func appendPageRow(rows [][]telego.InlineKeyboardButton, p Page, action func(index int) callback.Action) [][]telego.InlineKeyboardButton {
	var row []telego.InlineKeyboardButton

	if p.Index > 0 {
		row = append(row, button(texts.ButtonPrevPage, action(p.Index-1)))
	}

	if p.Index < p.Count-1 {
		row = append(row, button(texts.ButtonNextPage, action(p.Index+1)))
	}

	if len(row) == 0 {
		return rows
	}

	return append(rows, row)
}
