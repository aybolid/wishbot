package tgbot

import (
	"fmt"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type TextHandler func(textMsg *tgbotapi.Message) error

var textHandlers = map[string]TextHandler{
	"pendingGroupCreation": handleCreatingGroupFlow,
}

func handleText(textMsg *tgbotapi.Message) error {
	logger.SUGAR.Infow("handling text", "text", textMsg.Text, "chat_id", textMsg.Chat.ID, "from", textMsg.From)

	var err error

	if state.isPendingGroupCreation(textMsg.From.ID) {
		err = textHandlers["pendingGroupCreation"](textMsg)
	}

	return err
}

func handleCreatingGroupFlow(textMsg *tgbotapi.Message) error {
	group, err := db.CreateGroup(textMsg.From.ID, textMsg.Text)
	if err != nil {
		return err
	}

	state.releaseUser(textMsg.From.ID)

	resp := tgbotapi.NewMessage(textMsg.Chat.ID, fmt.Sprintf("Group \"%s\" was created!", group.Name))
	bot.HandledSend(resp)

	resp = tgbotapi.NewMessage(textMsg.Chat.ID, "Now you can add members to the group.")
	bot.HandledSend(resp)

	return nil
}
