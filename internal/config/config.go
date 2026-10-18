package config

import (
	"log/slog"
	"time"
)

type Config struct {
	Telegram Telegram `yaml:"telegram"`
	SQLite   SQLite   `yaml:"sqlite"`
	Render   Render   `yaml:"render"`
	Log      Log      `yaml:"log"`
}

type Telegram struct {
	Token      string        `validate:"required"      yaml:"token"`
	OwnerID    int64         `validate:"required,gt=0" yaml:"owner_id"`
	APITimeout time.Duration `validate:"required,gt=0" yaml:"api_timeout"`
	Retry      TelegramRetry `yaml:"retry"`
	Poll       TelegramPoll  `yaml:"poll"`
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
