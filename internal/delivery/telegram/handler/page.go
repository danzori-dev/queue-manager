package handler

import (
	"github.com/mymmrac/telego"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const pageSize = 8

type page struct {
	index int
	count int
	start int
	end   int
}

func paginate(total, requested int) page {
	count := max(1, (total+pageSize-1)/pageSize)
	index := min(max(requested, 0), count-1)
	start := index * pageSize

	return page{
		index: index,
		count: count,
		start: start,
		end:   min(start+pageSize, total),
	}
}

func appendPageRow(rows [][]telego.InlineKeyboardButton, p page, action func(index int) domain.Action) [][]telego.InlineKeyboardButton {
	var row []telego.InlineKeyboardButton

	if p.index > 0 {
		row = append(row, button(texts.ButtonPrevPage, action(p.index-1)))
	}

	if p.index < p.count-1 {
		row = append(row, button(texts.ButtonNextPage, action(p.index+1)))
	}

	if len(row) == 0 {
		return rows
	}

	return append(rows, row)
}
