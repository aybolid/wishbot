package tgbot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type callbackHandler = func(*tgbotapi.CallbackQuery) error

var callbackHandlers = map[string]callbackHandler{
	INVITE_MEMBER_CALLBACK_PREFIX: handleInviteMemberCallback,
	REJECT_INVITE_CALLBACK_PREFIX: handleRejectInviteCallback,
	ACCEPT_INVITE_CALLBACK_PREFIX: handleAcceptInviteCallback,
}

func handleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery) error {
	logger.Sugared.Infow("handling callback query", "data", callbackQuery.Data)

	delimIndex := strings.IndexByte(callbackQuery.Data, ':')
	prefix := callbackQuery.Data[0 : delimIndex+1]

	handler, ok := callbackHandlers[prefix]
	if ok {
		return handler(callbackQuery)
	}
	logger.Sugared.Errorw("no callback handler for prefix", "prefix", prefix)

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

	State.setPendingInviteCreation(userID, groupID)

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

	deleteReq := tgbotapi.NewDeleteMessage(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID)
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

func handleAcceptInviteCallback(callbackQuery *tgbotapi.CallbackQuery) error {
	inviterId, groupId, err := parseInviteCallbackQuery(callbackQuery, ACCEPT_INVITE_CALLBACK_PREFIX)
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

	_, err = db.CreateGroupMember(group.ID, callbackQuery.From.ID)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "You accepted the invite")
	bot.HandledSend(resp)

	deleteReq := tgbotapi.NewDeleteMessage(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID)
	bot.HandledSend(deleteReq)

	msg := tgbotapi.NewMessage(
		inviter.ChatID,
		fmt.Sprintf(
			"%s %s accepted your invite to the \"%s\" group.",
			callbackQuery.From.FirstName, callbackQuery.From.LastName,
			group.Name,
		),
	)
	bot.HandledSend(msg)

	return nil
}
