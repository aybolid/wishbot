package tgbot

import (
	"fmt"

	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CmdHandler func(api *TgBotAPI, cmdMsg *tgbotapi.Message) error

var cmdHandlers = map[string]CmdHandler{
	"/start": func(api *TgBotAPI, cmdMsg *tgbotapi.Message) error {
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, fmt.Sprintf("Hello, %s!", cmdMsg.From.FirstName))
		api.HandledSend(resp)

		resp = tgbotapi.NewMessage(cmdMsg.Chat.ID, "I am a bot that will help you with sharing your wishes with your friends.")
		api.HandledSend(resp)

		return nil
	},
}

func handleCommand(api *TgBotAPI, cmdMsg *tgbotapi.Message) error {
	logger.SUGAR.Infow("handling command", "command", cmdMsg.Text, "chat_id", cmdMsg.Chat.ID, "from", cmdMsg.From)

	var err error

	if handler, ok := cmdHandlers[cmdMsg.Text]; ok {
		err = handler(api, cmdMsg)
	} else {
		err = fmt.Errorf("unknown command received: %s", cmdMsg.Text)
	}

	return err
}
