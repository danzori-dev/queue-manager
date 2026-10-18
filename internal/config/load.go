package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/joho/godotenv"
)

const (
	defaultConfigPath = "configs/config.yml"
	defaultEnvPath    = ".env"
)

type loader struct {
	configPath string
	envPath    string
}

func Load() (Config, error) {
	cfg, err := load()
	if err != nil {
		return cfg, err
	}

	if err := validate(cfg); err != nil {
		return cfg, fmt.Errorf("%w: %w", ErrValidation, err)
	}

	return cfg, nil
}

func load() (Config, error) {
	var cfg Config

	l := loader{
		configPath: defaultConfigPath,
		envPath:    defaultEnvPath,
	}

	if err := godotenv.Load(l.envPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return cfg, fmt.Errorf("load env %s: %w", l.envPath, err)
	}

	raw, err := os.ReadFile(l.configPath)
	if err != nil {
		return cfg, fmt.Errorf("%w: %w", ErrReadFile, err)
	}

	var tree map[string]any
	if err := yaml.Unmarshal(raw, &tree); err != nil {
		return cfg, fmt.Errorf("%w: %w", ErrParseYAML, err)
	}

	if err := expand(tree); err != nil {
		return cfg, err
	}

	if err := decode(tree, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
