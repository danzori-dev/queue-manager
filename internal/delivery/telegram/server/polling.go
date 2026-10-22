package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mymmrac/telego"
)

func (s *Server) runPolling(ctx context.Context) error {
	for {
		err := s.pollOnce(ctx)
		if ctx.Err() != nil {
			return err
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

func (s *Server) pollOnce(parent context.Context) error {
	if err := s.deleteWebhook(parent); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	updates, err := s.bot.UpdatesViaLongPolling(
		ctx,
		&telego.GetUpdatesParams{
			Timeout:        s.cfg.PollTimeout,
			AllowedUpdates: allowedUpdates(),
		},
		telego.WithLongPollingRetryTimeout(s.cfg.RetryTimeout),
	)
	if err != nil {
		return fmt.Errorf("start long polling: %w", err)
	}

	return s.serve(parent, updates, nil)
}

func (s *Server) deleteWebhook(ctx context.Context) error {
	reqCtx, cancel := context.WithTimeout(ctx, s.cfg.APITimeout)
	defer cancel()

	if err := s.bot.DeleteWebhook(reqCtx, &telego.DeleteWebhookParams{DropPendingUpdates: false}); err != nil {
		return fmt.Errorf("delete webhook: %w", err)
	}

	return nil
}
