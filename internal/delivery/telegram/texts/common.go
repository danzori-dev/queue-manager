package texts

import "fmt"

const (
	InternalError     = "Что-то пошло не так, попробуй ещё раз."
	AccessDenied      = "Нет доступа."
	TooManyRequests   = "Не так быстро, попробуй чуть позже."
	StudentsNotLoaded = "Сначала загрузи список студентов: <code>/import_students</code>."
	NotWhitelisted    = "У тебя нет доступа к боту, обратись к администратору."
	UserUnreachable   = "Бот не может написать этому человеку: он ещё не запускал бота.\n" +
		"Попроси его открыть бота и нажать «Старт», затем отправь запрос снова."

	RequestsTooOften = "Слишком много запросов за последнее время. Подожди несколько минут."
	RequestCooldown  = "Этому человеку недавно уже был запрос от тебя. Подожди несколько минут."

	ButtonCancel   = "Отмена"
	ButtonPrevPage = "←"
	ButtonNextPage = "→"
)

func pageLine(index, count int) string {
	if count <= 1 {
		return ""
	}

	return fmt.Sprintf("\n\nСтраница %d из %d", index+1, count)
}
