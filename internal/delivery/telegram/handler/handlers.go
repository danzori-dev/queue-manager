package handler

import (
	"log/slog"
	"time"

	"github.com/mymmrac/telego"

	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler/activity"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler/admin"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler/carry"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler/general"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler/invite"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler/menu"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler/queue"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler/student"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler/swap"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/reply"
)

type QueueService interface {
	queue.QueueService
	menu.QueueService
	carry.QueueService
	general.QueueService
}

type StudentService interface {
	admin.StudentService
	queue.StudentService
	student.StudentService
}

type Deps struct {
	Bot        *telego.Bot
	APITimeout time.Duration
	Version    string
	Log        *slog.Logger

	Admins   admin.AdminService
	Queues   QueueService
	Students StudentService
	Swaps    swap.SwapService
	Invites  invite.InviteService
	Activity activity.Journal
}

type Handlers struct {
	General  *general.Handler
	Student  *student.Handler
	Admin    *admin.Handler
	Queue    *queue.Handler
	Menu     *menu.Handler
	Carry    *carry.Handler
	Swap     *swap.Handler
	Invite   *invite.Handler
	Activity *activity.Handler
}

func New(d Deps) Handlers {
	replier := reply.New(d.Bot, d.APITimeout, d.Log)

	swapHandler := swap.New(replier, d.Swaps)
	carryHandler := carry.New(replier, d.Queues)
	inviteHandler := invite.New(replier, d.Invites)

	return Handlers{
		General: general.New(
			general.Deps{
				Replier: replier,
				Queues:  d.Queues,
				Swaps:   swapHandler,
				Invites: inviteHandler,
				Version: d.Version,
				Log:     d.Log,
			},
		),
		Student: student.New(
			student.Deps{
				Replier:    replier,
				Students:   d.Students,
				Bot:        d.Bot,
				APITimeout: d.APITimeout,
			},
		),
		Admin:    admin.New(replier, d.Admins, d.Students),
		Queue:    queue.New(replier, d.Queues, d.Students),
		Menu:     menu.New(replier, d.Queues, carryHandler),
		Carry:    carryHandler,
		Swap:     swapHandler,
		Invite:   inviteHandler,
		Activity: activity.New(replier, d.Activity, d.Students),
	}
}
