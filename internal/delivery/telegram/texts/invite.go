package texts

import (
	"fmt"
	"html"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const (
	ButtonInviteAccept  = "Принять"
	ButtonInviteDecline = "Отклонить"
)

func InviteAsk(invite appqueue.Invite) string {
	return fmt.Sprintf(
		"%s зовёт тебя к себе на %d-е место в очереди «%s».\n\n"+
			"Если согласишься, вы будете стоять на одном месте.\n"+
			"Приглашение действует %d минут.",
		html.EscapeString(invite.FromName), invite.Position, html.EscapeString(invite.Event.Title),
		int(domain.InviteTTL.Minutes()),
	)
}
