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
	INVITE_MEMBER_CALLBACK_PREFIX:    handleInviteMemberCallback,
	REJECT_INVITE_CALLBACK_PREFIX:    handleRejectInviteCallback,
	ACCEPT_INVITE_CALLBACK_PREFIX:    handleAcceptInviteCallback,
	ADD_WISH_CALLBACK_PREFIX:         handleAddWishCallback,
	DISPLAY_WISHES_CALLBACK_PREFIX:   handleDisplayWishesCallback,
	LEAVE_GROUP_CALLBACK_PREFIX:      handleLeaveGroupCallback,
	ARE_YOU_SURE_NO_CALLBACK_PREFIX:  handleNo,
	ARE_YOU_SURE_YES_CALLBACK_PREFIX: handleYes,
}

func handleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery) error {
	defer func() {
		deleteReq := tgbotapi.NewDeleteMessage(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID)
		bot.HandledSend(deleteReq)
	}()

	logger.Sugared.Infow("handling callback query", "data", callbackQuery.Data)

	delimIndex := strings.IndexByte(callbackQuery.Data, ':')
	prefix := callbackQuery.Data[0 : delimIndex+1]
	logger.Sugared.Debugw("callback query prefix extracted", "prefix", prefix)

	handler, ok := callbackHandlers[prefix]
	if ok {
		return handler(callbackQuery)
	}
	logger.Sugared.Errorw("no callback handler for prefix", "prefix", prefix)

	return nil
}

func handleLeaveGroupCallback(callbackQuery *tgbotapi.CallbackQuery) error {
	groupID, err := strconv.ParseInt(callbackQuery.Data[len(LEAVE_GROUP_CALLBACK_PREFIX):], 10, 64)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}

	message := ""
	if group.OwnerID == callbackQuery.From.ID {
		message = fmt.Sprintf(
			"Do you really want to leave the \"%s\" group?\n<b>This action will delete the group, members and wishes as you are the owner.</b>",
			group.Name,
		)
	} else {
		message = fmt.Sprintf("Do you really want to leave the \"%s\" group?", group.Name)
	}

	sendAreYouSure(&areYouSureConfig{
		chatID:       callbackQuery.Message.Chat.ID,
		message:      message,
		actionID:     LEAVE_GROUP_ACTION,
		callbackData: fmt.Sprintf("%d", group.GroupID),
	})

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
	inviterID, groupID, err := parseInviteCallbackQuery(callbackQuery, REJECT_INVITE_CALLBACK_PREFIX)
	if err != nil {
		return err
	}
	inviter, err := db.GetUser(inviterID)
	if err != nil {
		return err
	}
	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "You rejected the invite")
	bot.HandledSend(resp)

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
	inviterID, groupID, err := parseInviteCallbackQuery(callbackQuery, ACCEPT_INVITE_CALLBACK_PREFIX)
	if err != nil {
		return err
	}
	inviter, err := db.GetUser(inviterID)
	if err != nil {
		return err
	}
	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}

	_, err = db.CreateGroupMember(group.GroupID, callbackQuery.From.ID)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "You accepted the invite")
	bot.HandledSend(resp)

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

func handleAddWishCallback(callbackQuery *tgbotapi.CallbackQuery) error {
	groupID, err := strconv.ParseInt(callbackQuery.Data[len(ADD_WISH_CALLBACK_PREFIX):], 10, 64)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(
		callbackQuery.Message.Chat.ID,
		fmt.Sprintf(
			"Ok! Lets add a wish to the \"%s\" group.",
			group.Name,
		),
	)
	bot.HandledSend(resp)

	resp = tgbotapi.NewMessage(
		callbackQuery.Message.Chat.ID,
		"Please send the URL of the wish you want to add with some description if applicable\\.\n\nExample:\n>>https://example\\.com\n>>This is a description",
	)
	resp.ParseMode = tgbotapi.ModeMarkdownV2
	bot.HandledSend(resp)

	State.setPendingWishCreation(callbackQuery.From.ID, groupID)

	return nil
}

func handleDisplayWishesCallback(callbackQuery *tgbotapi.CallbackQuery) error {
	groupID, err := strconv.ParseInt(callbackQuery.Data[len(DISPLAY_WISHES_CALLBACK_PREFIX):], 10, 64)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}

	wishes, err := db.GetGroupWishes(group.GroupID)
	if err != nil {
		return err
	}

	if len(wishes) == 0 {
		resp := tgbotapi.NewMessage(callbackQuery.Message.Chat.ID, "No wishes found for this group. /addwish")
		bot.HandledSend(resp)
		return nil
	}

	var groupedByUser = make(map[int64][]*db.Wish)
	for _, wish := range wishes {
		groupedByUser[wish.UserID] = append(groupedByUser[wish.UserID], wish)
	}

	resp := tgbotapi.NewMessage(
		callbackQuery.Message.Chat.ID,
		fmt.Sprintf(
			"Here are wishes from the \"%s\" group!",
			group.Name,
		),
	)
	bot.HandledSend(resp)

	for user, wishes := range groupedByUser {
		go func() {
			user, err := db.GetUser(user)
			if err != nil {
				logger.Sugared.Errorw("failed to get user for wishes display", "user_id", user, "err", err)
				return
			}

			text := ""

			if user.UserID == callbackQuery.From.ID {
				text += "Your wishes:\n\n"
			} else {
				text += fmt.Sprintf(
					"@%s wishes:\n\n",
					user.Username,
				)
			}

			for idx, wish := range wishes {
				text += fmt.Sprintf(
					"%d. %s\n%s\n\n",
					idx+1,
					wish.URL,
					wish.Description,
				)
			}

			resp := tgbotapi.NewMessage(
				callbackQuery.Message.Chat.ID,
				text,
			)
			bot.HandledSend(resp)
		}()
	}

	return nil
}
