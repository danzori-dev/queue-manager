package render

import (
	"github.com/mymmrac/telego"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/keyboard"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
)

type Messages struct {
	botUsername string
}

func NewMessages(botUsername string) *Messages {
	return &Messages{botUsername: botUsername}
}

func (m *Messages) Queue(view appqueue.View) (string, *telego.InlineKeyboardMarkup) {
	return texts.QueueMessage(view), keyboard.Queue(view, m.botUsername)
}

func (m *Messages) Removal(removal appqueue.Removal) string {
	return texts.Removal(removal)
}

func (m *Messages) SwapAsk(swap appqueue.Swap) (string, *telego.InlineKeyboardMarkup) {
	return texts.SwapAsk(swap), keyboard.SwapAnswer(swap.ID)
}
