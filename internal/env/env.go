package env

import (
	"os"
)

type vars struct {
	// Running mode (dev or prod).
	Mode string
	// Telegram bot API key.
	BotAPIKey string
}

var Vars *vars

// Loads environment variables from .env file (using joho/godotenv)
// and sets up the global VARS variable.
//
// Panics if the .env file is not found or if any of the required
// environment variables are not set.
func Init() {
	if Vars != nil {
		return
	}

	Vars = &vars{}

	Vars.Mode = os.Getenv("MODE")
	if Vars.Mode != "dev" && Vars.Mode != "prod" {
		Vars.Mode = "prod"
	}

	Vars.BotAPIKey = os.Getenv("BOT_API_KEY")
	if Vars.BotAPIKey == "" {
		panic("BOT_API_KEY is not set")
	}
}
