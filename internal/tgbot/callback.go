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
	if strings.HasPrefix(callbackQuery.Data, REJECT_INVITE_CALLBACK_PREFIX) {
		return handleRejectInviteCallback(callbackQuery)
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

	STATE.setPendingInviteCreation(userID, groupID)

	resp := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, fmt.Sprintf("Please mention the users you want to invite to the \"%s\" group.", group.Name))
	bot.HandledSend(resp)

	return nil
}

func handleRejectInviteCallback(callbackQuery *tgbotapi.CallbackQuery) error {
	inviterId, groupId, err := parseInviteCallbackQuery(callbackQuery, REJECT_INVITE_CALLBACK_PREFIX)
	if err != nil {
		return err
	}
	inviter, err := db.GetUser(inviterId)
	if err != nil {
		return err
	}
	group, err := db.GetGroup(groupId)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "You rejected the invite")
	bot.HandledSend(resp)

	deleteReq := tgbotapi.NewDeleteMessage(inviter.ChatID, callbackQuery.Message.MessageID)
	bot.HandledSend(deleteReq)

	msg := tgbotapi.NewMessage(
		inviter.ChatID,
		fmt.Sprintf(
			"%s %s rejected your invite to the \"%s\" group.",
			callbackQuery.From.FirstName, callbackQuery.From.LastName,
			group.Name,
		),
	)
	bot.HandledSend(msg)

	return nil
}
