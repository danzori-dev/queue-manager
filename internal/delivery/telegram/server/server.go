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

	select {
	case err := <-startErr:
		stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.cfg.StopTimeout)
		defer cancel()

		_ = bh.StopWithContext(stopCtx)

		if err == nil {
			err = errors.New("updates channel closed")
		}

		return fmt.Errorf("bot handler stopped: %w", err)

	case <-parent.Done():
		stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.cfg.StopTimeout)
		defer cancel()

		if err := bh.StopWithContext(stopCtx); err != nil && !errors.Is(err, context.Canceled) {
			s.log.WarnContext(
				ctx, "stop bot handler",
				slog.Any("err", err),
			)
		}

		s.log.InfoContext(ctx, "telegram server stopped")

		return nil
	}
}
