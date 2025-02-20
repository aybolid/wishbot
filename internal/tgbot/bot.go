package tgbot

import (
	"database/sql"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/env"
	"github.com/aybolid/wishbot/internal/locals"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type botAPI struct {
	*tgbotapi.BotAPI
}

type handleContext struct {
	user          *db.User
	localizer     *i18n.Localizer
	msg           *tgbotapi.Message
	callbackQuery *tgbotapi.CallbackQuery
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
		userID int64
	)

	if update.Message != nil {
		userID = update.Message.From.ID
		chatID = update.Message.Chat.ID
	}
	if update.CallbackQuery != nil {
		userID = update.CallbackQuery.From.ID
		chatID = update.CallbackQuery.Message.Chat.ID
	}

	user, err := db.GetUser(userID)
	if err != nil {
		if err == sql.ErrNoRows {
			var tgUser *tgbotapi.User
			if update.Message != nil {
				tgUser = update.Message.From
			} else if update.CallbackQuery != nil {
				tgUser = update.CallbackQuery.From
			}

			user, err = db.CreateUser(tgUser, chatID)
		}
	}

	ctx := &handleContext{
		user:          user,
		localizer:     locals.GetLocalizer(user.Language),
		msg:           update.Message,
		callbackQuery: update.CallbackQuery,
	}

	if err == nil {
		switch {
		case update.Message != nil:
			err = handleMessage(ctx)
		case update.CallbackQuery != nil:
			err = handleCallbackQuery(ctx)
		default:
			return
		}
	}

	if err != nil {
		logger.Sugared.Errorw("error handling update", "error", err)
		errResp := tgbotapi.NewMessage(chatID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "genericError",
			},
		))
		bot.HandledSend(errResp)
	}
}

// handleMessage processes incoming text messages.
func handleMessage(ctx *handleContext) error {
	logger.Sugared.Infow("received message",
		"text", ctx.msg.Text,
		"chat_id", ctx.msg.Chat.ID,
		"from", ctx.msg.From,
	)

	if ctx.msg.IsCommand() {
		return handleCommand(ctx)
	}
	return handleText(ctx)
}
