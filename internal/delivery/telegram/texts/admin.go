package texts

import (
	"fmt"
	"html"
	"strings"
)

const (
	AddAdminUsage = "Использование: <code>/addadmin &lt;user_id&gt;</code>"
	RmAdminUsage  = "Использование: <code>/rmadmin &lt;user_id&gt;</code>"

	NotAStudent        = "Этого пользователя нет в списке студентов."
	CannotAddSelf      = "Нельзя назначить администратором самого себя."
	CannotRemoveOwner  = "Нельзя удалить владельца бота."
	AdminAlreadyExists = "Этот пользователь уже администратор."
	AdminNotFound      = "Этот пользователь не администратор."
	AdminsEmpty        = "Список администраторов пуст."
)

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
	return fmt.Sprintf("Администратор %s удалён.", label)
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
