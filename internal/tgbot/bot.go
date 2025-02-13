package tgbot

import (
	"context"
	"log"
	"strings"

	"github.com/aybolid/wishbot/internal/env"
	"github.com/aybolid/wishbot/internal/tgbot/handlers"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var API *tgbotapi.BotAPI

// Initializes the Telegram bot API.
func Init() {
	bot, err := tgbotapi.NewBotAPI(env.VARS.BotAPIKey)
	if err != nil {
		panic(err)
	}
	API = bot

	API.Debug = env.VARS.Debug

	log.Printf("Authorized on account %s", API.Self.UserName)
}

// Listens to incoming Telegram updates.
//
// This function spaws a goroutine that listens to incoming updates and
// calls the appropriate handler. The returned cancel function can be used
// to stop the goroutine.
func ListenToUpdates() context.CancelFunc {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	updates := API.GetUpdatesChan(u)

	go receiveUpdates(ctx, updates)

	return cancel
}

func receiveUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel) {
	for {
		select {
		case <-ctx.Done():
			return
		case update := <-updates:
			handleUpdate(update)
		}
	}
}

func handleUpdate(update tgbotapi.Update) {
	switch {
	case update.Message != nil:
		handleMessage(update.Message)
	case update.CallbackQuery != nil:
		break
	}
}

func handleMessage(msg *tgbotapi.Message) {
	text := msg.Text

	log.Printf("[%s]: %s", msg.From.UserName, text)

	if strings.HasPrefix(text, "/") {
		// TODO: handle error
		_ = handlers.HandleCommand(API, msg)
	}
}
