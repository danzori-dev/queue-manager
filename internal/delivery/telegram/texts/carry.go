package texts

import (
	"fmt"
	"html"
	"strings"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
)

const (
	CarrySourcesEmpty = "Нет закрытых очередей для переноса."
	CarrySourceEmpty  = "В исходной очереди нет участников."
	CarryTargetsEmpty = "Нет открытых очередей для переноса."
	CarrySameEvent    = "Нельзя перенести людей в ту же очередь."
	CarrySourceOpen   = "Исходная очередь ещё открыта. Сначала закрой в ней запись."
	CarryTargetClosed = "Запись в очередь, куда переносишь, закрыта."
	CarryPosition     = "В исходной очереди нет такого места."
	CarryChanged      = "Состав очереди изменился. Проверь обновлённый предпросмотр и подтверди перенос ещё раз."

	ButtonCarry = "Перенести"

	CarryExpired    = "Меню переноса устарело. Отправь /carry ещё раз."
	CarryProcessing = "Предыдущее действие еще обрабатывается."
	CarryCanceled   = "Перенос отменён."
	CarryDoneAnswer = "Готово. Перенос выполнен."
)

func CarryChooseSource(page, pages int) string {
	return "Выбери закрытую очередь, из которой хочешь перенести участников" + pageLine(page, pages)
}

func CarryChoosePlace(title string, page, pages int) string {
	return fmt.Sprintf(
		"Очередь %s\n\nВыбери начальное место для переноса. "+
			"Будут перенесены все участники с выбранного места и всех следующих мест.",
		html.EscapeString(title),
	) + pageLine(page, pages)
}

func CarryChooseTarget(fromID int64, fromPos, page, pages int) string {
	return fmt.Sprintf(
		"Перенос из очереди #%d, начиная с %d-го места.\n\nВыбери открытую очередь назначения.",
		fromID, fromPos,
	) + pageLine(page, pages)
}

func CarryPlan(plan appqueue.CarryPlan, lines []appqueue.Line, page, pages int) string {
	var b strings.Builder

	fmt.Fprintf(
		&b, "Перенести %d чел. из «%s» в начало «%s»?\n",
		len(plan.Lines), html.EscapeString(plan.From.Title), html.EscapeString(plan.To.Title),
	)

	for _, place := range groupPlaces(lines) {
		names := make([]string, 0, len(place))

		for i, line := range place {
			names[i] = html.EscapeString(line.Name)
		}

		fmt.Fprintf(&b, "\n%d. %s", place[0].Position, strings.Join(names, ", "))
	}

	b.WriteString(pageLine(page, pages))

	return b.String()
}

func Carried(count int) string {
	return fmt.Sprintf("Перенесено человек: %d.", count)
}
