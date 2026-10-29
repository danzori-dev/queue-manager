package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"runtime/debug"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/danzori-dev/queue-manager/internal/application/activity"
	"github.com/danzori-dev/queue-manager/internal/application/admin"
	"github.com/danzori-dev/queue-manager/internal/application/queue"
	"github.com/danzori-dev/queue-manager/internal/application/student"
	"github.com/danzori-dev/queue-manager/internal/config"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/render"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/server"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/server/middleware"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/infrastructure/cache/memory"
	"github.com/danzori-dev/queue-manager/internal/infrastructure/client/telegram"
	logger "github.com/danzori-dev/queue-manager/internal/infrastructure/observability/logger/slog"
	"github.com/danzori-dev/queue-manager/internal/infrastructure/persistence/sqlite"
	"github.com/danzori-dev/queue-manager/internal/worker/cleaner"
	"github.com/danzori-dev/queue-manager/internal/worker/scheduler"
)

func Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		slog.ErrorContext(
			ctx, "load config",
			slog.Any("err", err),
		)

		return err
	}

	notifier, err := telegram.NewOwnerNotifier(cfg.Telegram.Token, cfg.Telegram.OwnerID, cfg.Telegram.APITimeout)
	if err != nil {
		slog.ErrorContext(
			ctx, "create owner notifier",
			slog.Any("err", err),
		)

		return err
	}

	log := logger.New(
		ctx, logger.Options{
			Level:   cfg.Log.Level,
			Format:  logger.Format(cfg.Log.Format),
			Version: version(),
			Secrets: []string{cfg.Telegram.Token, cfg.SQLite.AuthToken},
			Alerts:  notifier,
		},
	)

	if err := run(ctx, cfg, log, notifier); err != nil {
		log.ErrorContext(
			ctx, "application stopped",
			slog.Any("err", err),
		)

		return err
	}

	_ = notifier.Send(context.WithoutCancel(ctx), texts.BotStopped)

	return nil
}

func run(ctx context.Context, cfg config.Config, log *slog.Logger, notifier *telegram.OwnerNotifier) error {
	db, err := sqlite.Open(
		ctx,
		sqlite.Config{
			DSN:          cfg.SQLite.DSN,
			AuthToken:    cfg.SQLite.AuthToken,
			QueryTimeout: cfg.SQLite.QueryTimeout,
		},
	)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.WarnContext(
				ctx, "close database",
				slog.Any("err", closeErr),
			)
		}
	}()

	bot, err := telegram.NewBot(
		telegram.Params{
			Token: cfg.Telegram.Token,
			Retry: telegram.RetryConfig{
				MaxAttempts:  cfg.Telegram.Retry.MaxAttempts,
				ExponentBase: cfg.Telegram.Retry.ExponentBase,
				StartDelay:   cfg.Telegram.Retry.StartDelay,
				MaxDelay:     cfg.Telegram.Retry.MaxDelay,
			},
			HTTPTimeout: cfg.Telegram.Poll.Timeout + cfg.Telegram.APITimeout,
			Log:         log,
		},
	)
	if err != nil {
		return fmt.Errorf("new bot: %w", err)
	}

	meCtx, cancelMe := context.WithTimeout(ctx, cfg.Telegram.APITimeout)
	me, err := bot.GetMe(meCtx)

	cancelMe()

	if err != nil {
		return fmt.Errorf("telegram auth check: %w", err)
	}

	log.InfoContext(
		ctx, "telegram bot authorized",
		slog.String("username", me.Username),
	)

	adminRepo := sqlite.NewAdminRepository(db, cfg.SQLite.QueryTimeout)
	adminSvc := admin.NewService(adminRepo, cfg.Telegram.OwnerID)

	if err := adminSvc.Load(ctx); err != nil {
		return err
	}

	studentRepo := sqlite.NewStudentRepository(db, cfg.SQLite.QueryTimeout)
	studentSvc := student.NewService(studentRepo)

	if err := studentSvc.Load(ctx); err != nil {
		return err
	}

	if studentSvc.Count() == 0 {
		log.WarnContext(ctx, "student list is empty, send a JSON file with /import_students")
	}

	eventRepo := memory.NewEventCache(sqlite.NewEventRepository(db, cfg.SQLite.QueryTimeout))
	entryRepo := sqlite.NewEntryRepository(db, cfg.SQLite.QueryTimeout)
	targetRepo := memory.NewTargetCache(sqlite.NewTargetRepository(db, cfg.SQLite.QueryTimeout))

	queueMessenger := telegram.NewQueueMessenger(
		telegram.QueueMessengerDeps{
			Bot:        bot,
			Renderer:   render.NewMessages(me.Username),
			APITimeout: cfg.Telegram.APITimeout,
			Log:        log,
		},
	)

	renderScheduler := scheduler.New(
		scheduler.Config{
			Debounce:    cfg.Render.Debounce,
			MinInterval: cfg.Render.MinInterval,
			RetryDelay:  cfg.Render.RetryDelay,
			MaxAttempts: cfg.Render.MaxAttempts,
		}, log,
	)

	activityJournal := activity.New()

	queueSvc := queue.New(
		queue.Deps{
			Events:    eventRepo,
			Entries:   entryRepo,
			Targets:   targetRepo,
			Students:  studentSvc,
			Messages:  queueMessenger,
			Scheduler: renderScheduler,
			RequestLimit: queue.RequestLimit{
				Window:      cfg.RequestLimit.Window,
				MaxFailures: cfg.RequestLimit.MaxFailures,
				Pause:       cfg.RequestLimit.Pause,
			},
			Log:      log,
			Activity: activityJournal,
		},
	)

	srv := server.New(
		bot,
		server.Config{
			Mode:           server.Mode(cfg.Telegram.Mode),
			APITimeout:     cfg.Telegram.APITimeout,
			PollTimeout:    int(cfg.Telegram.Poll.Timeout.Seconds()),
			RetryTimeout:   cfg.Telegram.Poll.RetryTimeout,
			StopTimeout:    cfg.Telegram.Poll.StopTimeout,
			ReconnectDelay: cfg.Telegram.Poll.ReconnectDelay,
			Webhook: server.WebhookConfig{
				URL:    cfg.Telegram.Webhook.URL,
				Path:   cfg.Telegram.Webhook.Path,
				Listen: cfg.Telegram.Webhook.Listen,
			},
		},
		//nolint:contextcheck // ctx comes from each update
		server.RegisterRoutes(
			server.RouteParams{
				Bot: bot,
				Handlers: handler.New(
					handler.Deps{
						Bot:        bot,
						APITimeout: cfg.Telegram.APITimeout,
						Version:    version(),
						Log:        log,
						Admins:     adminSvc,
						Queues:     queueSvc,
						Students:   studentSvc,
						Swaps:      queueSvc,
						Invites:    queueSvc,
						Activity:   activityJournal,
					},
				),
				RoleResolver: adminSvc,
				Students:     studentSvc,
				RateLimit: server.RateLimit{
					Student: middleware.RateLimitConfig{
						Burst:    cfg.RateLimit.Student.Burst,
						Interval: cfg.RateLimit.Student.Interval,
					},
					Admin: middleware.RateLimitConfig{
						Burst:    cfg.RateLimit.Admin.Burst,
						Interval: cfg.RateLimit.Admin.Interval,
					},
				},
				APITimeout: cfg.Telegram.APITimeout,
				Log:        log,
			},
		),
		log,
	)

	_ = notifier.Send(ctx, texts.BotStarted(version()))

	cleanupWorker := cleaner.New(
		cleaner.Config{
			Retention: cfg.Cleanup.Retention,
			Interval:  cfg.Cleanup.Interval,
		}, log,
	)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(
		func() error {
			return renderScheduler.Run(ctx, queueSvc)
		},
	)

	g.Go(
		func() error {
			return srv.Run(ctx)
		},
	)

	g.Go(
		func() error {
			return cleanupWorker.Run(ctx, queueSvc)
		},
	)

	return g.Wait()
}

//nolint:gochecknoglobals // set at build time via -ldflags -X
var buildVersion string

func version() string {
	if buildVersion != "" {
		return buildVersion
	}

	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return "dev"
}
