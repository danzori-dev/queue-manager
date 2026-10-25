package config

import (
	"log/slog"
	"time"
)

type Config struct {
	Telegram     Telegram     `yaml:"telegram"`
	SQLite       SQLite       `yaml:"sqlite"`
	Render       Render       `yaml:"render"`
	Log          Log          `yaml:"log"`
	Cleanup      Cleanup      `yaml:"cleanup"`
	RateLimit    RateLimit    `yaml:"rate_limit"`
	RequestLimit RequestLimit `yaml:"request_limit"`
}

type Telegram struct {
	Token      string          `validate:"required"                       yaml:"token"`
	OwnerID    int64           `validate:"required,gt=0"                  yaml:"owner_id"`
	APITimeout time.Duration   `validate:"required,gt=0"                  yaml:"api_timeout"`
	Mode       string          `validate:"required,oneof=polling webhook" yaml:"mode"`
	Retry      TelegramRetry   `yaml:"retry"`
	Poll       TelegramPoll    `yaml:"poll"`
	Webhook    TelegramWebhook `yaml:"webhook"`
}

type TelegramRetry struct {
	MaxAttempts  int           `validate:"required,gte=1" yaml:"max_attempts"`
	ExponentBase float64       `validate:"required,gt=1"  yaml:"exponent_base"`
	StartDelay   time.Duration `validate:"required,gt=0"  yaml:"start_delay"`
	MaxDelay     time.Duration `validate:"required,gt=0"  yaml:"max_delay"`
}

type TelegramPoll struct {
	Timeout        time.Duration `validate:"required,gt=0" yaml:"timeout"`
	RetryTimeout   time.Duration `validate:"required,gt=0" yaml:"retry_timeout"`
	StopTimeout    time.Duration `validate:"required,gt=0" yaml:"stop_timeout"`
	ReconnectDelay time.Duration `validate:"required,gt=0" yaml:"reconnect_delay"`
}

type TelegramWebhook struct {
	URL    string `validate:"omitempty,https_url"    yaml:"url"`
	Path   string `validate:"required,startswith=/"  yaml:"path"`
	Listen string `validate:"required,hostname_port" yaml:"listen"`
}

type SQLite struct {
	DSN          string        `validate:"required"      yaml:"dsn"`
	AuthToken    string        `validate:"required"      yaml:"auth_token"`
	QueryTimeout time.Duration `validate:"required,gt=0" yaml:"query_timeout"`
}

type Render struct {
	Debounce    time.Duration `validate:"required,gt=0"  yaml:"debounce"`
	MinInterval time.Duration `validate:"required,gt=0"  yaml:"min_interval"`
	RetryDelay  time.Duration `validate:"required,gt=0"  yaml:"retry_delay"`
	MaxAttempts int           `validate:"required,gte=1" yaml:"max_attempts"`
}

type Log struct {
	Level  slog.Level `yaml:"level"`
	Format string     `validate:"required,oneof=json text" yaml:"format"`
}

type Cleanup struct {
	Retention time.Duration `validate:"required,gt=0" yaml:"retention"`
	Interval  time.Duration `validate:"required,gt=0" yaml:"interval"`
}

type RateLimit struct {
	Student RateLimitSettings `yaml:"student"`
	Admin   RateLimitSettings `yaml:"admin"`
}

type RateLimitSettings struct {
	Burst    int           `validate:"required,gte=1" yaml:"burst"`
	Interval time.Duration `validate:"required,gt=0"  yaml:"interval"`
}

type RequestLimit struct {
	Window      time.Duration `validate:"required,gt=0"  yaml:"window"`
	MaxFailures int           `validate:"required,gte=1" yaml:"max_failures"`
	Pause       time.Duration `validate:"required,gt=0"  yaml:"pause"`
}
