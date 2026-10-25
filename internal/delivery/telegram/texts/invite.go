package texts

import (
	"fmt"
	"html"

	appqueue "github.com/danzori-dev/queue-manager/internal/application/queue"
	domain "github.com/danzori-dev/queue-manager/internal/domain/queue"
)

const (
	InviteNotQueues   = "Ты не стоишь ни в одной открытой очереди."
	InviteChooseQueue = "Выбери очередь, в которой хочешь пригласить человека к себе:"
	InviteNobody      = "Некого пригласить: все студенты уже стоят с тобой."
	InviteUnavailable = "Этого студента сейчас нельзя пригласить."

	InviteNotInQueue = "Тебя уже нет в этой очереди."
	InviteSelf       = "Нельзя пригласить самого себя."
	AlreadyOnPlace   = "Этот человек уже стоит с тобой на одном месте."
	PlaceFull        = "На этом месте уже нет свободных мест."
	InviteBusy       = "У тебя или у этого человека уже есть активное приглашение."

	ButtonInviteSend    = "Отправить приглашение"
	ButtonInviteCancel  = "Отменить приглашение"
	ButtonInviteAccept  = "Принять"
	ButtonInviteDecline = "Отклонить"

	InviteAborted  = "Отменено."
	InviteNotYours = "Это не твоё приглашение."
	InviteGone     = "Приглашение уже неактуально."
	InviteChanged  = "Приглашение неактуально: пригласивший уже не стоит в очереди."
	InviteDeclined = "Приглашение отклонено."
	InviteCanceled = "Приглашение отменено."
)

func InviteChoosePerson(eventID int64, title string, position, page, pages int) string {
	return fmt.Sprintf(
		"<b>#%d %s</b>\nКого пригласить к себе на %d-е место?",
		eventID, html.EscapeString(title), position,
	) + pageLine(page, pages)
}

func InviteConfirm(name, title string, position int) string {
	return fmt.Sprintf(
		"Пригласить %s к себе на %d-е место в очереди «%s»?",
		html.EscapeString(name), position, html.EscapeString(title),
	)
}

func InviteSent(name string) string {
	return fmt.Sprintf(
		"Приглашение отправлено: %s. На ответ есть %d мин.",
		html.EscapeString(name), int(domain.InviteTTL.Minutes()),
	)
}

func InviteAsk(invite appqueue.Invite) string {
	return fmt.Sprintf(
		"%s приглашает тебя к себе на %d-е место в очереди «%s».\n\n"+
			"Если согласишься, вы будете стоять на одном месте.\n"+
			"Приглашение действует %d минут.",
		html.EscapeString(invite.FromName), invite.Position, html.EscapeString(invite.Event.Title),
		int(domain.InviteTTL.Minutes()),
	)
}

func InviteAcceptedForAddressee(title string, position int) string {
	return fmt.Sprintf(
		"Готово, теперь ты на %d-м месте в очереди «%s».",
		position, html.EscapeString(title),
	)
}

func InviteAcceptedForInitiator(name, title string, position int) string {
	return fmt.Sprintf(
		"%s теперь стоит с тобой на %d-м месте в очереди «%s».",
		html.EscapeString(name), position, html.EscapeString(title),
	)
}

func InviteDeclinedForInitiator(name, title string) string {
	return fmt.Sprintf(
		"%s отклоняет приглашение в очереди «%s».",
		html.EscapeString(name), html.EscapeString(title),
	)
}

func InviteFailedForInitiator(name, title string) string {
	return fmt.Sprintf(
		"Приглашение для %s в очереди «%s» не сработало: очередь изменилась. Отправь приглашение заново.",
		html.EscapeString(name), html.EscapeString(title),
	)
}
