package tgbot

import (
	"fmt"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func handleText(textMsg *tgbotapi.Message) error {
	logger.SUGAR.Infow("handling text", "text", textMsg.Text, "chat_id", textMsg.Chat.ID, "from", textMsg.From)

	var err error

	if state.isPendingGroupCreation(textMsg.From.ID) {
		err = handleCreatingGroupFlow(textMsg)
	}
	if state.isPendingInviteCreation(textMsg.From.ID) {
		err = handleCreatingInviteFlow(textMsg)
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

func handleCreatingInviteFlow(textMsg *tgbotapi.Message) error {
	return nil // TODO
}
