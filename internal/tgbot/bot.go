package tgbot

import (
	"context"

	"github.com/aybolid/wishbot/internal/env"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TgBotAPI struct {
	tgbotapi.BotAPI
}

// HandledSend is a wrapper around the Send method that logs sent messages and errors if any.
func (b *TgBotAPI) HandledSend(c tgbotapi.Chattable) {
	msg, err := b.Send(c)
	if err != nil {
		logger.SUGAR.Errorw("failed to send message", "error", err)
	} else {
		logger.SUGAR.Infow("sent message", "text", msg.Text, "chat_id", msg.Chat.ID)
	}
}

var bot *TgBotAPI

// Initializes the Telegram bot API.
//
// Panics if an error occurs.
func Init() {
	if bot != nil {
		return
	}

	botApi, err := tgbotapi.NewBotAPI(env.VARS.BotAPIKey)
	if err != nil {
		panic(err)
	}
	bot = &TgBotAPI{BotAPI: *botApi}

	bot.Debug = env.VARS.Debug

	logger.SUGAR.Infow("telegram bot initialized", "name", bot.Self.UserName)
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

	updates := bot.GetUpdatesChan(u)

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

	if msg.IsCommand() {
		err := handleCommand(msg)
		if err != nil {
			logger.SUGAR.Error(err)
			errResp := tgbotapi.NewMessage(msg.Chat.ID, "Oops, something went wrong. Please try again later.")
			bot.HandledSend(errResp)
		}
		return
	}

	handleText(msg)
}
