package handlers

import (
	"fmt"

	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CmdHandler func(api *tgbotapi.BotAPI, cmdMsg *tgbotapi.Message) error

var cmdHandlers = map[string]CmdHandler{
	"/start": func(api *tgbotapi.BotAPI, cmdMsg *tgbotapi.Message) error {
		return nil
	},
}

func HandleCommand(api *tgbotapi.BotAPI, cmdMsg *tgbotapi.Message) error {
	logger.SUGAR.Infow("handling command", "command", cmdMsg.Text, "chat_id", cmdMsg.Chat.ID, "from", cmdMsg.From)

	var err error

	if handler, ok := cmdHandlers[cmdMsg.Text]; ok {
		err = handler(api, cmdMsg)
	} else {
		err = fmt.Errorf("unknown command received: %s", cmdMsg.Text)
	}

	return err
}
