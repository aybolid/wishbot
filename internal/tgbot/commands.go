package tgbot

import (
	"fmt"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CmdHandler func(cmdMsg *tgbotapi.Message) error

var cmdHandlers = map[string]CmdHandler{
	"/start": func(cmdMsg *tgbotapi.Message) error {
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, fmt.Sprintf("Hello, %s!", cmdMsg.From.FirstName))
		bot.HandledSend(resp)

		resp = tgbotapi.NewMessage(cmdMsg.Chat.ID, "I am a bot that will help you with sharing your wishes with your friends.")
		bot.HandledSend(resp)

		return nil
	},
	"/creategroup": handleCreateGroup,
	"/mygroups":    handleMyGroups,
}

func handleCommand(cmdMsg *tgbotapi.Message) error {
	logger.SUGAR.Infow("handling command", "command", cmdMsg.Text, "chat_id", cmdMsg.Chat.ID, "from", cmdMsg.From)
	state.releaseUser(cmdMsg.From.ID)

	var err error

	if handler, ok := cmdHandlers[cmdMsg.Text]; ok {
		err = handler(cmdMsg)
	} else {
		logger.SUGAR.Errorw("unknown command received", "command", cmdMsg.Text, "chat_id", cmdMsg.Chat.ID, "from", cmdMsg.From)
	}

	return err
}

func handleCreateGroup(cmdMsg *tgbotapi.Message) error {
	userID := cmdMsg.From.ID
	state.setPendingGroupCreation(userID)

	resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "Please send the name for your new group")
	bot.HandledSend(resp)
	return nil
}

func handleMyGroups(cmdMsg *tgbotapi.Message) error {
	userID := cmdMsg.From.ID
	groups, err := db.GetUserGroups(userID)
	if err != nil {
		return err
	}

	for _, group := range groups {
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, fmt.Sprintf("Group: %s", group.Name))
		bot.HandledSend(resp)
	}

	return nil
}
