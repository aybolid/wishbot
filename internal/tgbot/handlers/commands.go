package handlers

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleCommand(api *tgbotapi.BotAPI, cmdMsg *tgbotapi.Message) error {
	var err error
	resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "")

	switch cmdMsg.Text {
	case "/start":
	}

	if resp.Text != "" {
		_, err = api.Send(resp)
	}

	return err
}

type CmdHandler func(resp *tgbotapi.MessageConfig) error
