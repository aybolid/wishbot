package tgbot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func handleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery) error {
	logger.SUGAR.Infow("handling callback query", "data", callbackQuery.Data)

	if strings.HasPrefix(callbackQuery.Data, INVITE_MEMBER_CALLBACK_PREFIX) {
		return handleInviteMemberCallback(callbackQuery)
	}

	return nil
}

func handleInviteMemberCallback(callbackQuery *tgbotapi.CallbackQuery) error {
	userID := callbackQuery.From.ID
	groupID, err := strconv.ParseInt(callbackQuery.Data[len(INVITE_MEMBER_CALLBACK_PREFIX):], 10, 64)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}

	state.setPendingInviteCreation(userID)

	resp := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, fmt.Sprintf("Please mention the users you want to invite to the \"%s\" group.", group.Name))
	bot.HandledSend(resp)

	return nil
}
