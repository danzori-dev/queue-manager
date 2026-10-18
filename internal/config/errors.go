package config

import "errors"

var (
	ErrReadFile     = errors.New("read config file")
	ErrParseYAML    = errors.New("parse config yaml")
	ErrDecodeConfig = errors.New("decode config")
	ErrValidation   = errors.New("validate config")

	ErrMissingReference = errors.New("missing reference")
	ErrInvalidReference = errors.New("invalid reference")
)
