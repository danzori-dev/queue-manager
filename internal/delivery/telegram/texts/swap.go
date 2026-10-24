package texts

import (
	"fmt"
	"html"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const (
	SwapNotQueues   = "Ты не стоишь ни в одной открытой очереди."
	SwapChooseQueue = "Выбери очередь, в которой хочешь поменяться местами:"
	SwapNobody      = "В этой очереди больше никого нет."

	SwapNotInQueue  = "Кого-то из вас уже нет в этой очереди."
	SwapUnreachable = "Бот не может написать этому человеку: он ещё не запускал бота.\n" +
		"Попроси его открыть бота и нажать «Старт», затем отправь запрос снова."
	SwapSelf = "Нельзя поменяться с самим собой."
	SwapBusy = "У тебя или у этого человека уже есть активный запрос на обмен."

	ButtonSwapSend    = "Отправить запрос"
	ButtonSwapCancel  = "Отменить запрос"
	ButtonSwapAccept  = "Принять"
	ButtonSwapDecline = "Отклонить"

	SwapAborted  = "Отменено."
	SwapNotYours = "Это не твой запрос."
	SwapGone     = "Запрос уже неактуален."
	SwapChanged  = "Очередь изменилась, обмен не выполнен."
	SwapDeclined = "Запрос отклонён."
	SwapCanceled = "Запрос отменён."
)

func SwapChoosePerson(eventID int64, title string, page, pages int) string {
	return fmt.Sprintf("<b>#%d %s</b>\nС кем поменяться?", eventID, html.EscapeString(title)) + pageLine(page, pages)
}

func SwapConfirm(title string, myPos int, otherName string, otherPos int) string {
	return fmt.Sprintf(
		"Обмен местами в очереди «%s»:\n\nТы - %d-е место\n%s - %d-е место\n\nОтправить запрос?",
		html.EscapeString(title), myPos, html.EscapeString(otherName), otherPos,
	)
}

func SwapSent(name string) string {
	return fmt.Sprintf(
		"Запрос отправлен: %s. На ответ есть 10 минут.",
		html.EscapeString(name),
	)
}

func SwapAcceptedForAddressee(newPos int) string {
	return fmt.Sprintf("Готово, вы поменялись местами.\nТвоё новое место: %d.", newPos)
}

func SwapAcceptedForInitiator(name, title string, newPos int) string {
	return fmt.Sprintf(
		"%s принимает обмен в очереди «%s».\nТвоё новое место: %d.",
		html.EscapeString(name), html.EscapeString(title), newPos,
	)
}

func SwapDeclinedForInitiator(name, title string) string {
	return fmt.Sprintf(
		"%s отклоняет обмен местами в очереди «%s».",
		html.EscapeString(name), html.EscapeString(title),
	)
}

func SwapFailedForInitiator(title string) string {
	return fmt.Sprintf(
		"Обмен в очереди «%s» не выполнен: очередь изменилась. Отправь запрос заново.",
		html.EscapeString(title),
	)
}

func SwapAsk(swap appqueue.Swap) string {
	return fmt.Sprintf(
		"%s предлагает поменяться с тобой местами в очереди «%s».\n\n"+
			"Твое место - %d\nМесто %s - %d\n\n"+
			"Запрос действует %d минут.",
		html.EscapeString(swap.FromName), html.EscapeString(swap.Event.Title),
		swap.ToPos, html.EscapeString(swap.FromName), swap.FromPos, int(domain.SwapTTL.Minutes()),
	)
}
