package env

import (
	"os"

	"github.com/joho/godotenv"
)

type vars struct {
	// Enables debug mode.
	Debug bool
	// Telegram bot API key.
	BotAPIKey string
}

var VARS = vars{}

// Loads environment variables from .env file (using joho/godotenv)
// and sets up the global VARS variable.
//
// Panics if the .env file is not found or if any of the required
// environment variables are not set.
func Init() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	VARS.Debug = os.Getenv("DEBUG") == "true"

	VARS.BotAPIKey = os.Getenv("BOT_API_KEY")
	if VARS.BotAPIKey == "" {
		panic("BOT_API_KEY is not set")
	}
}
