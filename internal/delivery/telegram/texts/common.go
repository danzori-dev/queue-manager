package texts

import "fmt"

const (
	InternalError     = "Что-то пошло не так, попробуй ещё раз."
	AccessDenied      = "Нет доступа."
	TooManyRequests   = "Не так быстро, попробуй чуть позже."
	StudentsNotLoaded = "Сначала загрузи список студентов: <code>/import_students</code>."

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
