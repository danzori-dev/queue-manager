package texts

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/danzori-dev/queue-manager/internal/application/activity"
	"github.com/danzori-dev/queue-manager/internal/application/queue"
)

const ButtonActivityToList = "← К списку"

func ActivityPeople(page, pages int) string {
	text := "<b>Активность студентов за последние 24 часа</b>"

	return text + pageLine(page, pages)
}

func ActivityEvents(name string, total int, events []activity.Event, page, pages int) string {
	text := fmt.Sprintf("<b>%s</b>\nДействий за 24 ч: %d", html.EscapeString(name), total)

	if len(events) == 0 {
		return text + "\n\nЗа последние 24 часа событий не было."
	}

	lines := make([]string, 0, len(events))

	for _, e := range events {
		lines = append(lines, fmt.Sprintf("<code>%s</code> %s", eventTime(e.At), eventText(e)))
	}

	return text + "\n\n" + strings.Join(lines, "\n") + pageLine(page, pages)
}

func eventTime(at time.Time) string {
	return at.Format("02.01 " + time.TimeOnly)
}

func eventText(e activity.Event) string {
	return strings.Join(eventParts(e), " | ")
}

func eventParts(e activity.Event) []string {
	values := make(map[string]string, len(e.Attrs))

	for _, a := range e.Attrs {
		values[a.Key] = html.EscapeString(a.Value.String())
	}

	parts := make([]string, 0, len(values)+1)
	parts = append(parts, e.Action)

	if id, ok := values[queue.AttrEventID]; ok {
		parts = append(parts, "Очередь #"+id)
	}

	if from, ok := values[queue.AttrFromEventID]; ok {
		parts = append(parts, fmt.Sprintf("Очередь #%s → #%s", from, values[queue.AttrToEventID]))
	}

	if title, ok := values[queue.AttrTitle]; ok {
		parts = append(parts, "«"+title+"»")
	}

	if position, ok := values[queue.AttrPosition]; ok {
		parts = append(parts, "Место "+position)
	}

	if count, ok := values[queue.AttrCount]; ok {
		parts = append(parts, count+" чел.")
	}

	if peer, ok := values[queue.AttrPeerName]; ok {
		parts = append(parts, "→ "+peer)
	}

	return parts
}
