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

type Config struct {
	APITimeout     time.Duration
	PollTimeout    int
	RetryTimeout   time.Duration
	StopTimeout    time.Duration
	ReconnectDelay time.Duration
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
	for {
		err := s.runOnce(ctx)
		if ctx.Err() != nil {
			return nil //nolint:nilerr // canceled ctx is a graceful shutdown, not a failure
		}

		s.log.ErrorContext(
			ctx, "polling loop stopped unexpectedly, reconnecting",
			slog.Duration("delay", s.cfg.ReconnectDelay),
			slog.Any("err", err),
		)

		timer := time.NewTimer(s.cfg.ReconnectDelay)
		select {
		case <-ctx.Done():
			timer.Stop()

			return nil

		case <-timer.C:
		}
	}
}

func (s *Server) runOnce(parent context.Context) error {
	if err := s.deleteWebhook(parent); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	updates, err := s.bot.UpdatesViaLongPolling(
		ctx,
		&telego.GetUpdatesParams{
			Timeout:        s.cfg.PollTimeout,
			AllowedUpdates: []string{"message", "callback_query", "my_chat_member"},
		},
		telego.WithLongPollingRetryTimeout(s.cfg.RetryTimeout),
	)
	if err != nil {
		return fmt.Errorf("start long polling: %w", err)
	}

	return s.serve(parent, updates)
}

func (s *Server) deleteWebhook(ctx context.Context) error {
	reqCtx, cancel := context.WithTimeout(ctx, s.cfg.APITimeout)
	defer cancel()

	if err := s.bot.DeleteWebhook(reqCtx, &telego.DeleteWebhookParams{DropPendingUpdates: false}); err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}

	return nil
}

func (s *Server) serve(ctx context.Context, updates <-chan telego.Update) error {
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
