package keyboard

import (
	"github.com/mymmrac/telego"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/callback"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
)

const (
	pageSize       = 8
	eventsPageSize = 20
)

type Page struct {
	Index int
	Count int
	start int
	end   int
}

func (p Page) Bounds() (start, end int) {
	return p.start, p.end
}

func PageOf(index int) int {
	return index / pageSize
}

func paginate(total, requested, size int) Page {
	count := max(1, (total+size-1)/size)
	index := min(max(requested, 0), count-1)
	start := index * size

	return Page{
		Index: index,
		Count: count,
		start: start,
		end:   min(start+size, total),
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
