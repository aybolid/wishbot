package tgbot

import (
	"github.com/aybolid/wishbot/internal/env"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type botAPI struct {
	*tgbotapi.BotAPI
}

// HandledSend is a wrapper around the Send method that logs sent messages and errors if any.
func (b *botAPI) HandledSend(c tgbotapi.Chattable) {
	msg, err := b.Send(c)
	if err != nil {
		logger.Sugared.Errorw("failed to send message", "error", err)
	} else {
		logger.Sugared.Infow("sent message", "text", msg.Text, "chat_id", msg.Chat.ID)
	}
}

var bot *botAPI

// Init initializes the Telegram bot API.
// It panics if an error occurs during initialization.
func Init() {
	if bot != nil {
		return
	}

	api, err := tgbotapi.NewBotAPI(env.Vars.BotAPIKey)
	if err != nil {
		panic(err)
	}

	bot = &botAPI{BotAPI: api}
	bot.Debug = env.Vars.Mode == env.DEV_MODE

	logger.Sugared.Infow("telegram bot initialized", "name", bot.Self.UserName)
}

// Listen starts receiving and processing incoming Telegram updates.
func Listen() {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := bot.GetUpdatesChan(updateConfig)
	for update := range updates {
		processUpdate(update)
	}
}

// processUpdate routes an incoming update to the appropriate handler.
func processUpdate(update tgbotapi.Update) {
	var (
		err    error
		chatID int64
	)

	switch {
	case update.Message != nil:
		chatID = update.Message.Chat.ID
		err = handleMessage(update.Message)
	case update.CallbackQuery != nil:
		chatID = update.CallbackQuery.Message.Chat.ID
		err = handleCallbackQuery(update.CallbackQuery)
	default:
		return
	}

	if err != nil {
		logger.Sugared.Errorw("error handling update", "error", err)
		errResp := tgbotapi.NewMessage(chatID, "Oops, something went wrong. Please try again later.")
		bot.HandledSend(errResp)
	}
}

// handleMessage processes incoming text messages.
func handleMessage(msg *tgbotapi.Message) error {
	logger.Sugared.Infow("received message",
		"text", msg.Text,
		"chat_id", msg.Chat.ID,
		"from", msg.From,
	)

	if msg.IsCommand() {
		return handleCommand(msg)
	}
	return handleText(msg)
}
