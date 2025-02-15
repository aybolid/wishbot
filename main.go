package main

import (
	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/env"
	"github.com/aybolid/wishbot/internal/logger"
	"github.com/aybolid/wishbot/internal/tgbot"
)

func init() {
	env.Init()
	logger.Init()

	db.Init()

	tgbot.Init()
}

func main() {
	tgbot.ListenToUpdates()
}
