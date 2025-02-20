package locals

import (
	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

const (
	UKRAINIAN = "uk"
	ENGLISH   = "en"
)

const (
	EN_LOCALS_FILE = "active.en.toml"
	UK_LOCALS_FILE = "active.uk.toml"
)

var bundle *i18n.Bundle

func Init() {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	bundle.MustLoadMessageFile(EN_LOCALS_FILE)
	bundle.MustLoadMessageFile(UK_LOCALS_FILE)
}

func GetLocalizer(lang string) *i18n.Localizer {
	return i18n.NewLocalizer(bundle, lang)
}
