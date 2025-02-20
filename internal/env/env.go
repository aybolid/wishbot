package env

import (
	"fmt"
	"os"
)

const (
	MODE_ENV    = "MODE"
	BOT_API_KEY = "BOT_API_KEY"
)

const (
	DEV_MODE     = "dev"
	PROD_MODE    = "prod"
	DEFAULT_MODE = PROD_MODE
)

type vars struct {
	// Running mode (dev or prod).
	Mode string
	// Telegram bot API key.
	BotAPIKey string
}

// Vars is the environment variables.
var Vars *vars

// Init initializes the environment variables.
func Init() {
	if Vars != nil {
		return
	}

	Vars = &vars{
		Mode:      os.Getenv(MODE_ENV),
		BotAPIKey: os.Getenv(BOT_API_KEY),
	}

	if Vars.Mode != DEV_MODE && Vars.Mode != PROD_MODE {
		Vars.Mode = DEFAULT_MODE
	}

	if Vars.BotAPIKey == "" {
		panic(fmt.Errorf("missing %s environment variable", BOT_API_KEY))
	}
}
