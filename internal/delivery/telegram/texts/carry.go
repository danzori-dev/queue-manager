package texts

import (
	"fmt"
	"html"
	"strings"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
)

const (
	CarryUsage = "Использование: <code>/carry &lt;откуда&gt; &lt;с_места&gt; &lt;куда&gt;</code>\n\n" +
		"Например, <code>/carry 1 7 2</code> перенесёт всех, начиная с 7-го места очереди #1, в начало очереди #2."

	CarrySameEvent    = "Нельзя перенести людей в ту же очередь."
	CarrySourceOpen   = "Исходная очередь ещё открыта. Сначала закрой в ней запись."
	CarryTargetClosed = "Запись в очередь, куда переносишь, закрыта."
	CarryPosition     = "В исходной очереди нет такого места."

	ButtonCarry = "Перенести"

	CarryExpired    = "Время на подтверждение вышло. Отправь /carry ещё раз."
	CarryCanceled   = "Перенос отменён."
	CarryDoneAnswer = "Готово. Перенос выполнен."
)

func CarryPlan(p appqueue.CarryPlan) string {
	var b strings.Builder

	fmt.Fprintf(
		&b, "Перенести %d чел. из «%s» в начало «%s»?\n",
		len(p.Lines), html.EscapeString(p.From.Title), html.EscapeString(p.To.Title),
	)

	for _, line := range p.Lines {
		fmt.Fprintf(&b, "\n%d. %s", line.Position, html.EscapeString(line.Name))
	}

	return b.String()
}

func Carried(count int) string {
	return fmt.Sprintf("Перенесено человек: %d.", count)
}
