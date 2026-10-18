package config

import (
	"fmt"

	"github.com/go-viper/mapstructure/v2"
)

func decode(tree map[string]any, cfg *Config) error {
	dec, err := mapstructure.NewDecoder(
		&mapstructure.DecoderConfig{
			Result:           cfg,
			TagName:          "yaml",
			WeaklyTypedInput: true,
			ErrorUnused:      true,
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.TextUnmarshallerHookFunc(),
			),
		},
	)
	if err != nil {
		return fmt.Errorf("build decoder: %w", err)
	}

	if err := dec.Decode(tree); err != nil {
		return fmt.Errorf("%w: %w", ErrDecodeConfig, err)
	}

	return nil
}
