package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"runtime/debug"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/danzori-dev/queue-manager/internal/application/admin"
	"github.com/danzori-dev/queue-manager/internal/application/queue"
	"github.com/danzori-dev/queue-manager/internal/application/student"
	"github.com/danzori-dev/queue-manager/internal/config"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/handler"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/server"
	"github.com/danzori-dev/queue-manager/internal/delivery/telegram/texts"
	"github.com/danzori-dev/queue-manager/internal/infrastructure/client/telegram"
	logger "github.com/danzori-dev/queue-manager/internal/infrastructure/observability/logger/slog"
	"github.com/danzori-dev/queue-manager/internal/infrastructure/persistence/sqlite"
	"github.com/danzori-dev/queue-manager/internal/worker/renderer"
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
		logger.Options{
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
	studentSvc := student.NewService(studentRepo, cfg.Telegram.OwnerID)

	if err := studentSvc.Load(ctx); err != nil {
		return err
	}

	if studentSvc.Count() == 0 {
		log.WarnContext(ctx, "student list is empty, send a JSON file with /import_students")
	}

	eventRepo := sqlite.NewEventRepository(db, cfg.SQLite.QueryTimeout)
	entryRepo := sqlite.NewEntryRepository(db, cfg.SQLite.QueryTimeout)
	targetRepo := sqlite.NewTargetRepository(db, cfg.SQLite.QueryTimeout)

	queueMessenger := telegram.NewQueueMessenger(
		telegram.QueueMessengerDeps{
			Bot:        bot,
			APITimeout: cfg.Telegram.APITimeout,
			Log:        log,
		},
	)

	scheduler := renderer.New(
		renderer.Config{
			Debounce:    cfg.Render.Debounce,
			MinInterval: cfg.Render.MinInterval,
			RetryDelay:  cfg.Render.RetryDelay,
			MaxAttempts: cfg.Render.MaxAttempts,
		}, log,
	)

	queueSvc := queue.New(
		queue.Deps{
			Events:    eventRepo,
			Entries:   entryRepo,
			Targets:   targetRepo,
			Students:  studentSvc,
			Messages:  queueMessenger,
			Scheduler: scheduler,
		},
	)

	tgHandler := handler.New(
		handler.Deps{
			Bot:        bot,
			Admin:      adminSvc,
			Students:   studentSvc,
			Queue:      queueSvc,
			Version:    version(),
			APITimeout: cfg.Telegram.APITimeout,
			Log:        log,
		},
	)

	srv := server.New(
		bot,
		server.Config{
			PollTimeout:    int(cfg.Telegram.Poll.Timeout.Seconds()),
			RetryTimeout:   cfg.Telegram.Poll.RetryTimeout,
			StopTimeout:    cfg.Telegram.Poll.StopTimeout,
			ReconnectDelay: cfg.Telegram.Poll.ReconnectDelay,
		},
		//nolint:contextcheck // ctx comes from each update
		server.RegisterRoutes(
			server.RouteParams{
				Bot:          bot,
				Handler:      tgHandler,
				RoleResolver: adminSvc,
				Students:     studentSvc,
				APITimeout:   cfg.Telegram.APITimeout,
				Log:          log,
			},
		),
		log,
	)

	_ = notifier.Send(ctx, texts.BotStarted(version()))

	g, ctx := errgroup.WithContext(ctx)

	g.Go(
		func() error {
			return scheduler.Run(ctx, queueSvc)
		},
	)

	g.Go(
		func() error {
			return srv.Run(ctx)
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
