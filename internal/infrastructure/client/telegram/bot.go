package telegram

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
)

type RetryConfig struct {
	MaxAttempts  int
	ExponentBase float64
	StartDelay   time.Duration
	MaxDelay     time.Duration
}

type Params struct {
	Token       string
	Retry       RetryConfig
	HTTPTimeout time.Duration
	Log         *slog.Logger
}

func NewBot(p Params) (*telego.Bot, error) {
	retryCaller := &ta.RetryCaller{
		Caller:       &ta.HTTPCaller{Client: &http.Client{Timeout: p.HTTPTimeout}},
		MaxAttempts:  p.Retry.MaxAttempts,
		ExponentBase: p.Retry.ExponentBase,
		StartDelay:   p.Retry.StartDelay,
		MaxDelay:     p.Retry.MaxDelay,
		RateLimit:    ta.RetryRateLimitAbort,
	}

	bot, err := telego.NewBot(
		p.Token,
		telego.WithAPICaller(retryCaller),
		telego.WithLogger(logger{log: p.Log}),
	)
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	return bot, nil
}
