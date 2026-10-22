package texts

import (
	"fmt"
	"html"
	"strings"
	"time"
)

const BotStopped = "🟡 Бот остановлен."

func BotStarted(version string) string {
	return fmt.Sprintf("🟢 Бот запущен.\nВерсия: %s", version)
}

type BotStatus struct {
	Version    string
	DBLatency  time.Duration
	DBErr      error
	OpenEvents int
}

func Status(s BotStatus) string {
	var b strings.Builder

	b.WriteString("<b>Состояние бота:</b>\n\n")
	fmt.Fprintf(&b, "Версия: <code>%s</code>\n", html.EscapeString(s.Version))

	if s.DBErr != nil {
		fmt.Fprintf(&b, "База: недоступна (<code>%s</code>)\n", html.EscapeString(s.DBErr.Error()))
	} else {
		fmt.Fprintf(&b, "База: отвечает, %d мс\n", s.DBLatency.Milliseconds())
		fmt.Fprintf(&b, "Открытых очередей: %d\n", s.OpenEvents)
	}

	return b.String()
}
