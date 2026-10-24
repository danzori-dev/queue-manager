package texts

import (
	"fmt"
	"html"
	"strings"
	"time"

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
	ButtonMembers       = "Участники"
	ButtonRename        = "Переименовать"
	ButtonDelete        = "Удалить очередь"
	ButtonConfirmDelete = "Да, удалить"
	ButtonBack          = "← К списку"
	ButtonBackToEvent   = "← К очереди"

	EventClosedAnswer = "Запись закрыта."
	EventOpenedAnswer = "Запись открыта."

	KickPressAgain      = "Нажми ещё раз в течение 5 секунд, чтобы убрать студента."
	ParticipantRemoved  = "Студент убран из очереди."
	ParticipantNotFound = "Этого студента уже нет в очереди."

	EventRenamed = "Название изменено."
	RenameUsage  = "Использование: <code>/rename &lt;id&gt; &lt;название&gt;</code>"
	TitleEmpty   = "Название не может быть пустым. Отправь другое."
	EventDeleted = "Очередь удалена."

	queueEmpty = "В очереди никого нет."
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
		b.WriteString("\n" + queueEmpty)

		return b.String()
	}

	for _, line := range view.Lines {
		fmt.Fprintf(&b, "\n%d. %s", line.Position, html.EscapeString(line.Name))

		if line.UserName != "" {
			fmt.Fprintf(&b, " (@%s)", line.UserName)
		}
	}

	return b.String()
}

func EventMembers(view appqueue.View, page, pages int) string {
	header := fmt.Sprintf("<b>#%d %s</b>\n", view.Event.ID, html.EscapeString(view.Event.Title))

	if len(view.Lines) == 0 {
		return header + queueEmpty
	}

	return header + "Нажми на студента, чтобы убрать его из очереди." + pageLine(page, pages)
}

func RenamePrompt(title string, ttl time.Duration) string {
	return fmt.Sprintf(
		"Отправь новое название для очереди «%s».\n\nНа ответ есть %d мин.",
		html.EscapeString(title), int(ttl.Minutes()),
	)
}

func DeleteConfirm(title string) string {
	return fmt.Sprintf(
		"Удалить очередь <b>%s</b>?\n\nСообщение очереди в группе и список участников будут удалены.",
		html.EscapeString(title),
	)
}
