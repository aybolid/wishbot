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
	var mentions []tgbotapi.MessageEntity
	for _, entity := range textMsg.Entities {
		if entity.Type == "mention" {
			mentions = append(mentions, entity)
		}
	}

	if len(mentions) == 0 {
		resp := tgbotapi.NewMessage(textMsg.Chat.ID, "Please mention at least one user to invite.")
		bot.HandledSend(resp)
		return nil
	}

	for _, mention := range mentions {
		var user *db.User

		if mention.User != nil {
			var err error
			user, err = db.GetUser(mention.User.ID)
			if err != nil {
				resp := tgbotapi.NewMessage(
					textMsg.Chat.ID,
					fmt.Sprintf(
						"Seems like %s %s didn't chat with me yet. Please try again after they do.", mention.User.FirstName, mention.User.LastName,
					),
				)
				bot.HandledSend(resp)
				continue
			}
		} else {
			// + 1 to skip the @ symbol
			userName := textMsg.Text[mention.Offset+1 : mention.Offset+mention.Length]
			logger.SUGAR.Debugw("extracted user name from text", "username", userName)

			var err error
			user, err = db.GetUserByUsername(userName)
			if err != nil {
				resp := tgbotapi.NewMessage(textMsg.Chat.ID, fmt.Sprintf("Seems like %s didn't chat with me yet. Please try again after they do.", userName))
				bot.HandledSend(resp)
				continue
			}
		}

		inviteMsg := tgbotapi.NewMessage(user.ChatID, fmt.Sprintf("Your are invitied! %s", user.Username))
		_, err := bot.Send(inviteMsg)
		if err != nil {
			resp := tgbotapi.NewMessage(textMsg.Chat.ID, fmt.Sprintf("Something went wrong while inviting %s. Please try again later.", user.Username))
			bot.HandledSend(resp)
		} else {
			logger.SUGAR.Infow("invited user", "user_id", user.UserID, "chat_id", user.ChatID)
			resp := tgbotapi.NewMessage(textMsg.Chat.ID, fmt.Sprintf("Invited %s", user.Username))
			bot.HandledSend(resp)
		}
	}

	return nil
}
