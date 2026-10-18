package texts

import (
	"fmt"
	"html"
	"strings"

	"github.com/danzori-dev/queue-manager/internal/domain/student"
)

const (
	Welcome = "Привет! Я <b>QueueManager</b> - бот для управления очередями.\n\n" +
		"Используй /help, чтобы посмотреть список доступных команд."

	HelpCommon = "<b>Команды:</b>\n\n" +
		"<code>/help</code> - показать эту справку"
	HelpAdmin = "<b>Администратор:</b>\n\n" +
		"<code>/settopic</code> - назначить топик, в котором будут публиковаться очереди (написать в самом топике)\n" +
		"<code>/new &lt;название&gt;</code> - создать очередь"
	HelpOwner = "<b>Владелец:</b>\n\n" +
		"<code>/addadmin &lt;user_id&gt;</code> - назначить администратора\n" +
		"<code>/rmadmin &lt;user_id&gt;</code> - удалить администратора\n" +
		"<code>/admins</code> - список администраторов\n" +
		"<code>/students</code> - список студентов\n" +
		"<code>/import_students</code> - загрузить список студентов"

	InternalError = "Что-то пошло не так, попробуй ещё раз."
	AccessDenied  = "Нет доступа."

	StudentsNotLoaded = "Сначала загрузите список студентов: <code>/import_students</code>."

	AddAdminUsage      = "Использование: <code>/addadmin &lt;user_id&gt;</code>"
	RmAdminUsage       = "Использование: <code>/rmadmin &lt;user_id&gt;</code>"
	ImportStudentUsage = `Пришлите JSON-файл документом с подписью <code>/import_students</code>.`
	NewEventUsage      = "Использование: <code>/new &lt;title&gt;</code>"

	NotAStudent        = "Такого пользователя нет в списке студентов."
	CannotAddSelf      = "Нельзя назначить администратором самого себя."
	CannotRemoveOwner  = "Нельзя удалить владельца бота."
	AdminAlreadyExists = "Этот пользователь уже является администратором."
	AdminNotFound      = "Такого администратора нет в списке."
	AdminsEmpty        = "Список администраторов пуст."

	EventTitleTooLong = "Название слишком длинное: максимум 100 символов."
	EventCreated      = "Очередь создана и опубликована."
	QueueTopicSet     = "Готово: очереди будут публиковаться в этом топике."
	QueueTopicChanged = "Топик для очереди изменен: новые очереди будут публиковаться здесь."
	QueueTopicNotSet  = "Сначала выполните <code>/settopic</code> в топике группы, куда публиковать очереди."

	LeftQueue     = "Ты вышел из очереди."
	NotInQueue    = "Тебя нет в этой очереди."
	QueueClosed   = "Запись в эту очередь закрыта."
	QueueNotFound = "Очередь не найдена."

	StudentsListEmpty = "Список студентов пуст.\n\n" + ImportStudentUsage
)

func Help(sections ...string) string {
	parts := make([]string, 0, len(sections))

	for _, s := range sections {
		if s != "" {
			parts = append(parts, s)
		}
	}

	return strings.Join(parts, "\n\n")
}

func UserLabel(userID int64, name string) string {
	if name == "" {
		return fmt.Sprintf("<code>%d</code>", userID)
	}

	return fmt.Sprintf("%s (<code>%d</code>)", html.EscapeString(name), userID)
}

func AdminAdded(label string) string {
	return fmt.Sprintf("Администратор %s добавлен.", label)
}

func AdminRemoved(label string) string {
	return fmt.Sprintf("Администратор %s удален.", label)
}

func AdminList(labels []string) string {
	if len(labels) == 0 {
		return AdminsEmpty
	}

	var b strings.Builder

	b.WriteString("<b>Администраторы бота:</b>")

	for i, label := range labels {
		fmt.Fprintf(&b, "\n%d. %s", i+1, label)
	}

	return b.String()
}

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
	return "Список не загружен: <code>" + html.EscapeString(err.Error()) + "</code>"
}

func JoinedQueue(position int) string {
	return fmt.Sprintf("Ты в очереди: %d-й", position)
}

func AlreadyInQueue(position int) string {
	return fmt.Sprintf("Ты уже в очереди: %d-й", position)
}
