package texts

import (
	"fmt"
	"html"
	"strings"

	"github.com/danzori-dev/queue-manager/internal/domain/student"
)

const (
	ImportStudentUsage = "Пришли JSON-файл документом с подписью <code>/import_students</code>."
	StudentsListEmpty  = "Список студентов пуст.\n\n" + ImportStudentUsage
)

func StudentList(students []student.Student) string {
	if len(students) == 0 {
		return StudentsListEmpty
	}

	var b strings.Builder

	fmt.Fprintf(&b, "<b>Студенты (%d):</b>", len(students))

	for i, s := range students {
		fmt.Fprintf(&b, "\n%d. %s - <code>%d</code>", i+1, html.EscapeString(s.Name), s.ID)
	}

	return b.String()
}

func StudentsImported(total int) string {
	return fmt.Sprintf("Загружено студентов: <b>%d</b>.", total)
}

func StudentsImportFailed(err error) string {
	return fmt.Sprintf("Список не загружен:\n<code>%s</code>", html.EscapeString(err.Error()))
}
