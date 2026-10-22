package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/mymmrac/telego"
	th "github.com/mymmrac/telego/telegohandler"
)

type Mode string

const (
	ModePolling Mode = "polling"
	ModeWebhook Mode = "webhook"
)

type Config struct {
	Mode           Mode
	APITimeout     time.Duration
	PollTimeout    int
	RetryTimeout   time.Duration
	StopTimeout    time.Duration
	ReconnectDelay time.Duration
	Webhook        WebhookConfig
}

type Register func(bh *th.BotHandler)

type Server struct {
	bot      *telego.Bot
	cfg      Config
	register Register
	log      *slog.Logger
}

func New(bot *telego.Bot, cfg Config, register Register, log *slog.Logger) *Server {
	return &Server{
		bot:      bot,
		cfg:      cfg,
		register: register,
		log:      log,
	}
}

func (s *Server) Run(ctx context.Context) error {
	var err error

	if s.cfg.Mode != ModeWebhook {
		err = s.runPolling(ctx)
	} else {
		err = s.runWebhook(ctx)
	}

	if ctx.Err() != nil {
		return nil //nolint:nilerr // canceled ctx is a graceful shutdown, not a failure
	}

	return err
}

func (s *Server) serve(ctx context.Context, updates <-chan telego.Update, beforeStop func(stopCtx context.Context)) error {
	bh, err := th.NewBotHandler(s.bot, updates)
	if err != nil {
		return fmt.Errorf("create bot handler: %w", err)
	}

	s.register(bh)

	startErr := make(chan error, 1)

	go func() {
		startErr <- bh.Start()
	}()

	s.log.InfoContext(ctx, "telegram server started")

	stop := func() error {
		stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.cfg.StopTimeout)
		defer cancel()

		if beforeStop != nil {
			beforeStop(stopCtx)
		}

		return bh.StopWithContext(stopCtx)
	}

	select {
	case err := <-startErr:
		_ = stop()

		if err == nil {
			err = errors.New("updates channel closed")
		}

		return fmt.Errorf("bot handler stopped: %w", err)

	case <-ctx.Done():
		if err := stop(); err != nil && !errors.Is(err, context.Canceled) {
			s.log.WarnContext(
				ctx, "stop bot handler",
				slog.Any("err", err),
			)
		}

		s.log.InfoContext(ctx, "telegram server stopped")

		return nil
	}
}

func allowedUpdates() []string {
	return []string{"message", "callback_query", "my_chat_member"}
}
