package texts

import (
	"fmt"
	"math"
	"time"
)

const (
	InternalError     = "Что-то пошло не так, попробуй ещё раз."
	AccessDenied      = "Нет доступа."
	TooManyRequests   = "Не так быстро, попробуй чуть позже."
	StudentsNotLoaded = "Сначала загрузи список студентов: <code>/import_students</code>."
	NotWhitelisted    = "У тебя нет доступа к боту, обратись к администратору."
	UserUnreachable   = "Бот не может написать этому человеку: он ещё не запускал бота.\n" +
		"Попроси его открыть бота и нажать «Старт», затем отправь запрос снова."

	ButtonCancel   = "Отмена"
	ButtonPrevPage = "←"
	ButtonNextPage = "→"
)

func RequestsTooOften(wait time.Duration) string {
	return fmt.Sprintf("Слишком много запросов за последнее время. Попробуй снова через %s.", waitText(wait))
}

func RequestCooldown(wait time.Duration) string {
	return fmt.Sprintf("Этому человеку недавно уже был запрос от тебя. Попробуй снова через %s.", waitText(wait))
}

func waitText(wait time.Duration) string {
	return fmt.Sprintf("%d мин", int(math.Ceil(wait.Minutes())))
}

func pageLine(index, count int) string {
	if count <= 1 {
		return ""
	}

	return fmt.Sprintf("\n\nСтраница %d из %d", index+1, count)
}
