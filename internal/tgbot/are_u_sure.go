package tgbot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	ARE_YOU_SURE_YES_CALLBACK_PREFIX = "surey:"
	ARE_YOU_SURE_NO_CALLBACK_PREFIX  = "suren:"
)

const (
	LEAVE_GROUP_ACTION = iota
	DELETE_WISH_ACTION
	KICK_MEMBER_ACTION
)

type areYouSureConfig struct {
	chatID       int64
	message      string
	actionID     int
	callbackData string
}

type actionHandler = func(int, *tgbotapi.CallbackQuery) error

var actionHandlers = map[int]actionHandler{
	LEAVE_GROUP_ACTION: handleGroupLeave,
	DELETE_WISH_ACTION: handleDeleteWish,
	KICK_MEMBER_ACTION: handleKickMember,
}

func sendAreYouSure(config *areYouSureConfig) error {
	text := fmt.Sprintf("<b>Are you sure?</b>\n\n%s", config.message)
	msg := tgbotapi.NewMessage(config.chatID, text)

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("No", ARE_YOU_SURE_NO_CALLBACK_PREFIX),
			tgbotapi.NewInlineKeyboardButtonData(
				"Yes",
				fmt.Sprintf(
					"%s%d:%s",
					ARE_YOU_SURE_YES_CALLBACK_PREFIX, config.actionID, config.callbackData,
				),
			),
		),
	)
	msg.ParseMode = tgbotapi.ModeHTML

	bot.HandledSend(msg)

	return nil
}

func handleNo(callbackQuery *tgbotapi.CallbackQuery) error {
	return nil
}

func handleYes(callbackQuery *tgbotapi.CallbackQuery) error {
	payload := strings.Split(callbackQuery.Data[len(ARE_YOU_SURE_YES_CALLBACK_PREFIX):], ":")

	logger.Sugared.Debugw("are you sure payload", "payload", payload)

	actionID, err := strconv.ParseInt(payload[0], 10, 64)
	if err != nil {
		return err
	}

	handler, ok := actionHandlers[int(actionID)]
	if ok {
		return handler(len(ARE_YOU_SURE_YES_CALLBACK_PREFIX)+len(payload[0])+1, callbackQuery)
	} else {
		logger.Sugared.Errorw("no action handler for action id", "action_id", actionID)
	}

	return nil
}

func handleKickMember(dataOffset int, callbackQuery *tgbotapi.CallbackQuery) error {
	payload := strings.Split(callbackQuery.Data[dataOffset:], ":")
	logger.Sugared.Debugw("kick member payload", "payload", payload)

	userID, err := strconv.ParseInt(payload[0], 10, 64)
	if err != nil {
		return err
	}
	groupID, err := strconv.ParseInt(payload[1], 10, 64)
	if err != nil {
		return err
	}

	user, err := db.GetUser(userID)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}

	err = db.DeleteGroupMember(groupID, userID)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, fmt.Sprintf("You kicked @%s from the \"%s\" group.", user.Username, group.Name))
	bot.HandledSend(resp)

	go func() {
		msg := tgbotapi.NewMessage(
			user.ChatID,
			fmt.Sprintf(
				"Hey! You were kicked from the \"%s\" group.",
				group.Name,
			),
		)
		bot.HandledSend(msg)
	}()

	return nil
}

func handleDeleteWish(dataOffset int, callbackQuery *tgbotapi.CallbackQuery) error {
	wishID, err := strconv.ParseInt(callbackQuery.Data[dataOffset:], 10, 64)
	if err != nil {
		return err
	}

	err = db.DeleteWish(wishID)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "Wish deleted.")
	bot.HandledSend(resp)

	return nil
}

func handleGroupLeave(dataOffset int, callbackQuery *tgbotapi.CallbackQuery) error {
	groupID, err := strconv.ParseInt(callbackQuery.Data[dataOffset:], 10, 64)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}
	members, err := db.GetGroupMembers(groupID)
	if err != nil {
		return err
	}

	err = db.DeleteGroupMember(groupID, callbackQuery.From.ID)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, fmt.Sprintf("You left the \"%s\" group.", group.Name))
	bot.HandledSend(resp)

	if group.OwnerID == callbackQuery.From.ID {
		resp = tgbotapi.NewMessage(
			callbackQuery.Message.Chat.ID,
			"As you are the owner, the group and all related data was deleted. I hope you didn't do it accidentally.",
		)
		bot.HandledSend(resp)

		for _, member := range members {
			if member.UserID == callbackQuery.From.ID {
				continue
			}
			go func() {
				user, err := db.GetUser(member.UserID)
				if err != nil {
					logger.Sugared.Errorw("error getting user for notification", "user_id", member.UserID, "error", err)
					return
				}

				msg := tgbotapi.NewMessage(
					user.ChatID,
					fmt.Sprintf(
						"Hey! The \"%s\" group was deleted by owner.",
						group.Name,
					),
				)

				bot.HandledSend(msg)
			}()
		}
	} else {
		for _, member := range members {
			if member.UserID == callbackQuery.From.ID {
				continue
			}
			go func() {
				user, err := db.GetUser(member.UserID)
				if err != nil {
					logger.Sugared.Errorw("error getting user for notification", "user_id", member.UserID, "error", err)
					return
				}

				msg := tgbotapi.NewMessage(
					user.ChatID,
					fmt.Sprintf(
						"Hey! %s left the \"%s\" group.",
						callbackQuery.From.FirstName,
						group.Name,
					),
				)

				bot.HandledSend(msg)
			}()
		}
	}

	return nil
}
