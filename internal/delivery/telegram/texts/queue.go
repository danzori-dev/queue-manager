package texts

import (
	"fmt"
	"html"
	"strings"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const (
	QueueTopicSet     = "Готово. Очереди будут публиковаться в этом топике."
	QueueTopicChanged = "Готово. Теперь новые очереди будут публиковаться в этом топике."
	QueueTopicNotSet  = "Сначала отправь <code>/settopic</code> в топике группы, где будут публиковаться очереди."

	NewEventUsage     = "Использование: <code>/new &lt;название&gt;</code>"
	EventTitleTooLong = "Название слишком длинное: не больше 100 символов."
	EventCreated      = "Очередь создана и опубликована."

	ButtonJoin   = "Встать в очередь"
	ButtonLeave  = "Выйти"
	ButtonSkip   = "Пропустить место"
	ButtonSwap   = "Поменяться местами"
	ButtonInvite = "Пригласить к себе"

	QueueClosed   = "Запись в эту очередь закрыта."
	QueueNotFound = "Очередь не найдена."
	LeftQueue     = "Тебя больше нет в очереди."
	NotInQueue    = "Тебя нет в этой очереди."

	SkipPressAgain = "Нажми ещё раз в течение 5 секунд, чтобы пропустить одно место."
	AlreadyLast    = "Ты и так в конце очереди."
)

func JoinedQueue(position int) string {
	return fmt.Sprintf("Твоё место в очереди: %d", position)
}

func AlreadyInQueue(position int) string {
	return fmt.Sprintf("Ты уже в очереди, твоё место: %d", position)
}

func Skipped(position int) string {
	return fmt.Sprintf("Теперь твоё место: %d", position)
}

func QueueMessage(view appqueue.View) string {
	var b strings.Builder

	fmt.Fprintf(&b, "<b>#%d %s</b>\n", view.Event.ID, html.EscapeString(view.Event.Title))

	if view.Event.Status == domain.StatusClosed {
		b.WriteString("🔒 Запись закрыта.\n")
	}

	if len(view.Lines) == 0 {
		b.WriteString("\nВ очереди никого нет.")

		return b.String()
	}

	places := groupPlaces(view.Lines)
	writePlaces(&b, places)

	return b.String()
}

func Removal(r appqueue.Removal) string {
	actor := "Администратор"

	if r.ActorName != "" {
		actor += " " + html.EscapeString(r.ActorName)
	}

	return fmt.Sprintf(
		`%s убрал <a href="tg://user?id=%d">%s</a> из очереди «%s».`,
		actor, r.UserID, html.EscapeString(r.UserName), html.EscapeString(r.Event.Title),
	)
}

func groupPlaces(lines []appqueue.Line) [][]appqueue.Line {
	var places [][]appqueue.Line

	for i, line := range lines {
		if i == 0 || line.Position != lines[i-1].Position {
			places = append(places, nil)
		}

		places[len(places)-1] = append(places[len(places)-1], line)
	}

	return places
}

func writePlaces(b *strings.Builder, places [][]appqueue.Line) {
	for _, place := range places {
		people := make([]string, 0, len(place))

		for _, line := range place {
			person := html.EscapeString(line.Name)
			if line.UserName != "" {
				person += fmt.Sprintf(" (@%s)", line.UserName)
			}

			people = append(people, person)
		}

		fmt.Fprintf(b, "\n%d. %s", place[0].Position, strings.Join(people, ", "))
	}
}
