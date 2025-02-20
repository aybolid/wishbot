package tgbot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type callbackHandler = func(*handleContext) error

var callbackHandlers = map[string]callbackHandler{
	INVITE_MEMBER_CALLBACK_PREFIX:    handleInviteMemberCallback,
	REJECT_INVITE_CALLBACK_PREFIX:    handleRejectInviteCallback,
	ACCEPT_INVITE_CALLBACK_PREFIX:    handleAcceptInviteCallback,
	ADD_WISH_CALLBACK_PREFIX:         handleAddWishCallback,
	DISPLAY_WISHES_CALLBACK_PREFIX:   handleDisplayWishesCallback,
	LEAVE_GROUP_CALLBACK_PREFIX:      handleLeaveGroupCallback,
	ARE_YOU_SURE_NO_CALLBACK_PREFIX:  handleNo,
	ARE_YOU_SURE_YES_CALLBACK_PREFIX: handleYes,
	DELETE_WISH_CALLBACK_PREFIX:      handleDeleteWishCallback,
	MANAGE_WISHES_CALLBACK_PREFIX:    handleManageWishesCallback,
	MANAGE_MEMBERS_CALLBACK_PREFIX:   handleManageMembersCallback,
	KICK_MEMBER_CALLBACK_PREFIX:      handleKickMemberCallback,
}

func handleCallbackQuery(ctx *handleContext) error {
	defer func() {
		deleteReq := tgbotapi.NewDeleteMessage(ctx.callbackQuery.Message.Chat.ID, ctx.callbackQuery.Message.MessageID)
		bot.HandledSend(deleteReq)
	}()

	logger.Sugared.Infow("handling callback query", "data", ctx.callbackQuery.Data)

	delimIndex := strings.IndexByte(ctx.callbackQuery.Data, ':')
	prefix := ctx.callbackQuery.Data[0 : delimIndex+1]
	logger.Sugared.Debugw("callback query prefix extracted", "prefix", prefix)

	handler, ok := callbackHandlers[prefix]
	if ok {
		return handler(ctx)
	}
	logger.Sugared.Errorw("no callback handler for prefix", "prefix", prefix)

	return nil
}

func handleKickMemberCallback(ctx *handleContext) error {
	payload := strings.Split(ctx.callbackQuery.Data[len(KICK_MEMBER_CALLBACK_PREFIX):], ":")
	logger.Sugared.Debugw("kick member payload", "payload", payload)

	userID, err := strconv.ParseInt(payload[0], 10, 64)
	if err != nil {
		return err
	}
	groupID, err := strconv.ParseInt(payload[1], 10, 64)
	if err != nil {
		return err
	}

	member, err := db.GetGroupMember(groupID, userID)
	if err != nil {
		return err
	}
	user, err := db.GetUser(member.UserID)
	if err != nil {
		return err
	}
	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}

	err = sendAreYouSure(
		&areYouSureConfig{
			chatID:       ctx.callbackQuery.Message.Chat.ID,
			message:      fmt.Sprintf("Are you sure you want to kick @%s from the \"%s\" group?", user.Username, group.Name),
			actionID:     KICK_MEMBER_ACTION,
			callbackData: fmt.Sprintf("%d:%d", user.UserID, groupID),
		},
	)

	return err
}

const KICK_MEMBER_CALLBACK_PREFIX = "kick_member:"

func handleManageMembersCallback(ctx *handleContext) error {
	groupID, err := strconv.ParseInt(ctx.callbackQuery.Data[len(MANAGE_MEMBERS_CALLBACK_PREFIX):], 10, 64)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}
	if group.OwnerID != ctx.callbackQuery.From.ID {
		logger.Sugared.Errorw("not the owner of the group", "group_id", groupID, "owner_id", group.OwnerID, "user_id", ctx.callbackQuery.From.ID)
		resp := tgbotapi.NewMessage(ctx.callbackQuery.Message.Chat.ID, "You are not the owner of this group.")
		bot.HandledSend(resp)
		return nil
	}

	members, err := db.GetGroupMembers(groupID)
	if err != nil {
		return err
	}

	filteredMembers := make([]*db.GroupMember, 0)
	for _, member := range members {
		if member.UserID == ctx.callbackQuery.From.ID {
			continue
		}
		filteredMembers = append(filteredMembers, member)
	}

	if len(filteredMembers) == 0 {
		resp := tgbotapi.NewMessage(ctx.callbackQuery.Message.Chat.ID, fmt.Sprintf("No members to manage found for the \"%s\" group.", group.Name))
		bot.HandledSend(resp)
		return nil
	}

	resp := tgbotapi.NewMessage(
		ctx.callbackQuery.Message.Chat.ID,
		fmt.Sprintf(
			"Here are the members of the \"%s\" group.",
			group.Name,
		),
	)
	bot.HandledSend(resp)

	for _, member := range filteredMembers {
		go func() {
			user, err := db.GetUser(member.UserID)
			if err != nil {
				logger.Sugared.Errorw("failed to get user for member display", "user_id", member.UserID, "err", err)
				return
			}

			userWishes, err := db.GetUserWishes(member.UserID, groupID)
			if err != nil {
				logger.Sugared.Errorw("failed to get user wishes for member display", "user_id", member.UserID, "err", err)
				return
			}

			msg := tgbotapi.NewMessage(
				ctx.callbackQuery.Message.Chat.ID,
				fmt.Sprintf(
					"@%s\nThey have %d wishes.",
					user.Username,
					len(userWishes),
				),
			)

			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Kick", fmt.Sprintf("%s%d:%d", KICK_MEMBER_CALLBACK_PREFIX, member.UserID, groupID)),
				),
			)

			bot.HandledSend(msg)
		}()
	}

	return nil
}

func handleManageWishesCallback(ctx *handleContext) error {
	groupID, err := strconv.ParseInt(ctx.callbackQuery.Data[len(MANAGE_WISHES_CALLBACK_PREFIX):], 10, 64)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}
	wishes, err := db.GetUserWishes(ctx.callbackQuery.From.ID, groupID)
	if err != nil {
		return err
	}

	if len(wishes) == 0 {
		resp := tgbotapi.NewMessage(
			ctx.callbackQuery.Message.Chat.ID, fmt.Sprintf("No wishes found for the \"%s\" group. /addwish", group.Name),
		)
		bot.HandledSend(resp)
		return nil
	}

	resp := tgbotapi.NewMessage(
		ctx.callbackQuery.Message.Chat.ID,
		fmt.Sprintf(
			"Here are your wishes from the \"%s\" group.",
			group.Name,
		),
	)
	bot.HandledSend(resp)

	for _, wish := range wishes {
		go func() {
			msg := tgbotapi.NewMessage(
				ctx.callbackQuery.Message.Chat.ID,
				fmt.Sprintf(
					"%s\n%s\n\n",
					wish.URL,
					wish.Description,
				),
			)

			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Delete", fmt.Sprintf("%s%d", DELETE_WISH_CALLBACK_PREFIX, wish.WishID)),
				),
			)

			bot.HandledSend(msg)
		}()
	}

	return nil
}

func handleDeleteWishCallback(ctx *handleContext) error {
	wishId, err := strconv.ParseInt(ctx.callbackQuery.Data[len(DELETE_WISH_CALLBACK_PREFIX):], 10, 64)
	if err != nil {
		return err
	}

	wish, err := db.GetWish(wishId)
	if err != nil {
		return err
	}

	wishText := fmt.Sprintf(
		"%s\n%s",
		wish.URL,
		wish.Description,
	)

	err = sendAreYouSure(&areYouSureConfig{
		chatID:       ctx.callbackQuery.Message.Chat.ID,
		message:      fmt.Sprintf("Are you sure you want to delete this wish?\n\n%s", wishText),
		actionID:     DELETE_WISH_ACTION,
		callbackData: fmt.Sprintf("%d", wish.WishID),
	})

	return err
}

func handleLeaveGroupCallback(ctx *handleContext) error {
	groupID, err := strconv.ParseInt(ctx.callbackQuery.Data[len(LEAVE_GROUP_CALLBACK_PREFIX):], 10, 64)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}

	message := ""
	if group.OwnerID == ctx.callbackQuery.From.ID {
		message = fmt.Sprintf(
			"Do you really want to leave the \"%s\" group?\n<b>This action will delete the group, members and wishes as you are the owner.</b>",
			group.Name,
		)
	} else {
		message = fmt.Sprintf("Do you really want to leave the \"%s\" group?", group.Name)
	}

	err = sendAreYouSure(&areYouSureConfig{
		chatID:       ctx.callbackQuery.Message.Chat.ID,
		message:      message,
		actionID:     LEAVE_GROUP_ACTION,
		callbackData: fmt.Sprintf("%d", group.GroupID),
	})

	return err
}

func handleInviteMemberCallback(ctx *handleContext) error {
	userID := ctx.callbackQuery.From.ID
	groupID, err := strconv.ParseInt(ctx.callbackQuery.Data[len(INVITE_MEMBER_CALLBACK_PREFIX):], 10, 64)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}

	State.setPendingInviteCreation(userID, groupID)

	resp := tgbotapi.NewMessage(ctx.callbackQuery.Message.Chat.ID, fmt.Sprintf("Please mention the users you want to invite to the \"%s\" group.", group.Name))
	bot.HandledSend(resp)

	return nil
}

func handleRejectInviteCallback(ctx *handleContext) error {
	inviterID, groupID, err := parseInviteCallbackQuery(ctx.callbackQuery, REJECT_INVITE_CALLBACK_PREFIX)
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

	resp := tgbotapi.NewMessage(ctx.callbackQuery.Message.Chat.ID, "You rejected the invite")
	bot.HandledSend(resp)

	msg := tgbotapi.NewMessage(
		inviter.ChatID,
		fmt.Sprintf(
			"%s %s rejected your invite to the \"%s\" group.",
			ctx.callbackQuery.From.FirstName, ctx.callbackQuery.From.LastName,
			group.Name,
		),
	)
	bot.HandledSend(msg)

	return nil
}

func handleAcceptInviteCallback(ctx *handleContext) error {
	inviterID, groupID, err := parseInviteCallbackQuery(ctx.callbackQuery, ACCEPT_INVITE_CALLBACK_PREFIX)
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

	_, err = db.CreateGroupMember(group.GroupID, ctx.callbackQuery.From.ID)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(ctx.callbackQuery.Message.Chat.ID, "You accepted the invite")
	bot.HandledSend(resp)

	msg := tgbotapi.NewMessage(
		inviter.ChatID,
		fmt.Sprintf(
			"%s %s accepted your invite to the \"%s\" group.",
			ctx.callbackQuery.From.FirstName, ctx.callbackQuery.From.LastName,
			group.Name,
		),
	)
	bot.HandledSend(msg)

	return nil
}

func handleAddWishCallback(ctx *handleContext) error {
	groupID, err := strconv.ParseInt(ctx.callbackQuery.Data[len(ADD_WISH_CALLBACK_PREFIX):], 10, 64)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(
		ctx.callbackQuery.Message.Chat.ID,
		fmt.Sprintf(
			"Ok! Lets add a wish to the \"%s\" group.",
			group.Name,
		),
	)
	bot.HandledSend(resp)

	resp = tgbotapi.NewMessage(
		ctx.callbackQuery.Message.Chat.ID,
		"Please send the URL of the wish you want to add with some description if applicable\\.\n\nExample:\n>>https://example\\.com\n>>This is a description",
	)
	resp.ParseMode = tgbotapi.ModeMarkdownV2
	bot.HandledSend(resp)

	State.setPendingWishCreation(ctx.callbackQuery.From.ID, groupID)

	return nil
}

func handleDisplayWishesCallback(ctx *handleContext) error {
	groupID, err := strconv.ParseInt(ctx.callbackQuery.Data[len(DISPLAY_WISHES_CALLBACK_PREFIX):], 10, 64)
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
		resp := tgbotapi.NewMessage(ctx.callbackQuery.Message.Chat.ID, "No wishes found for this group. /addwish")
		bot.HandledSend(resp)
		return nil
	}

	var groupedByUser = make(map[int64][]*db.Wish)
	for _, wish := range wishes {
		groupedByUser[wish.UserID] = append(groupedByUser[wish.UserID], wish)
	}

	resp := tgbotapi.NewMessage(
		ctx.callbackQuery.Message.Chat.ID,
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

			if user.UserID == ctx.callbackQuery.From.ID {
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
				ctx.callbackQuery.Message.Chat.ID,
				text,
			)
			bot.HandledSend(resp)
		}()
	}

	return nil
}
