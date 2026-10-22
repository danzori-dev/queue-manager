package server

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/mymmrac/telego"
)

const webhookReadHeaderTimeout = 10 * time.Second

type WebhookConfig struct {
	URL    string
	Path   string
	Listen string
}

func (s *Server) runWebhook(parent context.Context) error {
	webhookURL, err := url.JoinPath(s.cfg.Webhook.URL, s.cfg.Webhook.Path)
	if err != nil {
		return fmt.Errorf("build webhook url: %w", err)
	}

	var lc net.ListenConfig

	ln, err := lc.Listen(parent, "tcp", s.cfg.Webhook.Listen)
	if err != nil {
		return fmt.Errorf("listen webhook: %w", err)
	}

	secret := rand.Text()

	httpSrv := &http.Server{
		ReadHeaderTimeout: webhookReadHeaderTimeout,
	}

	updatesCtx, cancelUpdates := context.WithCancel(context.WithoutCancel(parent))
	defer cancelUpdates()

	setCtx, cancelSet := context.WithTimeout(parent, s.cfg.APITimeout)
	defer cancelSet()

	updates, err := s.bot.UpdatesViaWebhook(
		updatesCtx, telego.WebhookHTTPServer(httpSrv, s.cfg.Webhook.Path, secret),
		telego.WithWebhookSet(
			setCtx, &telego.SetWebhookParams{
				URL:            webhookURL,
				AllowedUpdates: allowedUpdates(),
				SecretToken:    secret,
			},
		),
	)
	if err != nil {
		_ = ln.Close()

		return fmt.Errorf("start webhook: %w", err)
	}

	ctx, cancel := context.WithCancelCause(parent)
	defer cancel(nil)

	go func() {
		if err := httpSrv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
			cancel(fmt.Errorf("serve webhook: %w", err))
		}
	}()

	s.log.InfoContext(
		ctx, "telegram webhook set",
		slog.String("url", webhookURL),
		slog.String("listen", s.cfg.Webhook.Listen),
	)

	shutdown := func(stopCtx context.Context) {
		if err := httpSrv.Shutdown(stopCtx); err != nil {
			s.log.WarnContext(
				ctx, "shutdown webhook server",
				slog.Any("err", err),
			)

			_ = httpSrv.Close()
		}
	}

	if err := s.serve(ctx, updates, shutdown); err != nil {
		return err
	}

	return context.Cause(ctx)
}
