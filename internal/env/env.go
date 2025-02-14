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
	// Path to the SQLite database.
	DBPath string
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

	err := godotenv.Load()
	if err != nil {
		panic(err)
	}

	Vars = &vars{}

	Vars.Debug = os.Getenv("DEBUG") == "true"

	Vars.BotAPIKey = os.Getenv("BOT_API_KEY")
	if Vars.BotAPIKey == "" {
		panic("BOT_API_KEY is not set")
	}

	Vars.DBPath = os.Getenv("DB_PATH")
	if Vars.DBPath == "" {
		panic("DB_PATH is not set")
	}
}
