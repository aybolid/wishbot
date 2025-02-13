package tgbot

import (
	"context"
	"strings"

	"github.com/aybolid/wishbot/internal/env"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TgBotAPI struct {
	tgbotapi.BotAPI
}

func (b *TgBotAPI) HandledSend(c tgbotapi.Chattable) {
	msg, err := b.Send(c)
	if err != nil {
		logger.SUGAR.Errorw("failed to send message", "error", err)
	} else {
		logger.SUGAR.Infow("sent message", "text", msg.Text, "chat_id", msg.Chat.ID)
	}
}

var API *TgBotAPI

// Initializes the Telegram bot API.
//
// Panics if an error occurs.
func Init() {
	if API != nil {
		return
	}

	bot, err := tgbotapi.NewBotAPI(env.VARS.BotAPIKey)
	if err != nil {
		panic(err)
	}
	API = &TgBotAPI{BotAPI: *bot}

	API.Debug = env.VARS.Debug

	logger.SUGAR.Infow("telegram bot initialized", "name", API.Self.UserName)
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

	logger.SUGAR.Infow("received message", "text", text, "chat_id", msg.Chat.ID, "from", msg.From)

	if strings.HasPrefix(text, "/") {
		err := handleCommand(API, msg)
		if err != nil {
			logger.SUGAR.Error(err)
		}
		return
	}
}
