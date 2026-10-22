package texts

import (
	"fmt"
	"html"
	"strings"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const (
	EventsTitle = "<b>Очереди</b>\nВыбери очередь, чтобы управлять ей."
	EventsEmpty = "Очередей нет."
	EventGone   = "Эта очередь уже удалена."

	ButtonClose         = "Закрыть запись"
	ButtonOpen          = "Открыть запись"
	ButtonRefresh       = "Обновить"
	ButtonDelete        = "Удалить очередь"
	ButtonConfirmDelete = "Да, удалить"
	ButtonBack          = "← К списку"

	EventClosedAnswer = "Запись закрыта."
	EventOpenedAnswer = "Запись открыта."

	ParticipantRemoved  = "Студент убран из очереди."
	ParticipantNotFound = "Этого студента уже нет в очереди."

	EventDeleted = "Очередь удалена."
)

func EventMenu(view appqueue.View) string {
	var b strings.Builder

	fmt.Fprintf(&b, "<b>#%d %s</b>\n", view.Event.ID, html.EscapeString(view.Event.Title))

	if view.Event.Status == domain.StatusClosed {
		b.WriteString("Статус: 🔒 запись закрыта\n")
	} else {
		b.WriteString("Статус: запись открыта\n")
	}

	if len(view.Lines) == 0 {
		b.WriteString("\nВ очереди никого нет.")

		return b.String()
	}

	for _, line := range view.Lines {
		fmt.Fprintf(&b, "\n%d. %s", line.Position, html.EscapeString(line.Name))

		if line.UserName != "" {
			fmt.Fprintf(&b, " (@%s)", line.UserName)
		}
	}

	b.WriteString("\n\nНажми на студента, чтобы убрать его из очереди.")

	return b.String()
}

func DeleteConfirm(title string) string {
	return fmt.Sprintf(
		"Удалить очередь <b>%s</b>?\n\nСообщение очереди в группе и список участников будут удалены.",
		html.EscapeString(title),
	)
}
