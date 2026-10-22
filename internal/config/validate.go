package config

import (
	"reflect"

	"github.com/go-playground/validator/v10"
)

const modeWebhook = "webhook"

func validate(cfg any) error {
	v := validator.New(validator.WithRequiredStructEnabled())

	v.RegisterTagNameFunc(
		func(f reflect.StructField) string {
			return f.Tag.Get("yaml")
		},
	)

	v.RegisterStructValidation(validateTelegram, Telegram{})

	return v.Struct(cfg)
}

func validateTelegram(sl validator.StructLevel) {
	tg, ok := reflect.TypeAssert[Telegram](sl.Current())
	if !ok || tg.Mode != modeWebhook {
		return
	}

	if tg.Webhook.URL == "" {
		sl.ReportError(tg.Webhook.URL, "webhook.url", "Webhook.URL", "required_if", "mode webhook")
	}
}
