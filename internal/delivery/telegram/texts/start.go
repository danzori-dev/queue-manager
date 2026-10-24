package texts

import "strings"

const (
	Welcome = "Привет! Я <b>QueueManager</b> - бот для управления очередями.\n\n" +
		"Используй /help, чтобы посмотреть список доступных команд."

	HelpCommon = "<b>Команды:</b>\n\n" +
		"<code>/help</code> - показать эту справку\n" +
		"<code>/swap</code> - предложить поменяться местами в очереди"
	HelpAdmin = "<b>Администратор:</b>\n\n" +
		"<code>/settopic</code> - назначить топик, в котором будут публиковаться очереди (написать в самом топике)\n" +
		"<code>/new &lt;название&gt;</code> - создать очередь\n" +
		"<code>/rename &lt;id&gt; &lt;название&gt;</code> - переименовать очередь\n" +
		"<code>/events</code> - управление очередями\n" +
		"<code>/carry &lt;откуда&gt; &lt;с_места&gt; &lt;куда&gt;</code> - перенести людей из одной закрытой очереди в начало другой"
	HelpOwner = "<b>Владелец:</b>\n\n" +
		"<code>/status</code> - состояние бота\n" +
		"<code>/addadmin &lt;user_id&gt;</code> - назначить администратора\n" +
		"<code>/rmadmin &lt;user_id&gt;</code> - удалить администратора\n" +
		"<code>/admins</code> - список администраторов\n" +
		"<code>/students</code> - список студентов\n" +
		"<code>/import_students</code> - загрузить список студентов"
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
