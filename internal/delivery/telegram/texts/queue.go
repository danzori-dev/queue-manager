package texts

import "fmt"

const (
	QueueTopicSet     = "Готово. Очереди будут публиковаться в этом топике."
	QueueTopicChanged = "Готово. Теперь новые очереди будут публиковаться в этом топике."
	QueueTopicNotSet  = "Сначала отправь <code>/settopic</code> в топике группы, где будут публиковаться очереди."

	NewEventUsage     = "Использование: <code>/new &lt;название&gt;</code>"
	EventTitleTooLong = "Название слишком длинное: не больше 100 символов."
	EventCreated      = "Очередь создана и опубликована."

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
