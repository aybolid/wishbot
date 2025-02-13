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
		if !cmdMsg.From.IsBot {
			_, err := db.CreateUser(cmdMsg.From, cmdMsg.Chat.ID)
			if err != nil {
				return err
			}
		}

		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, fmt.Sprintf("Hello, %s!", cmdMsg.From.FirstName))
		bot.HandledSend(resp)

		resp = tgbotapi.NewMessage(cmdMsg.Chat.ID, "I am a bot that will help you with sharing your wishes with your friends.")
		bot.HandledSend(resp)

		return nil
	},
	"/creategroup": handleCreateGroup,
	"/mygroups":    handleMyGroups,
	"/addmember":   handleAddMember,
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

const INVITE_MEMBER_CALLBACK_PREFIX = "invite_member:"

func handleAddMember(cmdMsg *tgbotapi.Message) error {
	groups, err := db.GetOwnedGroups(cmdMsg.From.ID)
	if err != nil {
		return err
	}

	if len(groups) == 0 {
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "You don't have any created groups yet. Please create one first.")
		bot.HandledSend(resp)
		return nil
	}

	resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "<b>Invite another member.</b>\n\nSelect a group to add a member to (you can add members only to groups you created).")

	var buttons []tgbotapi.InlineKeyboardButton
	for _, group := range groups {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(group.Name, fmt.Sprintf("%s%d", INVITE_MEMBER_CALLBACK_PREFIX, group.ID)))
	}

	markup := tgbotapi.NewInlineKeyboardMarkup(buttons)

	resp.ReplyMarkup = markup
	resp.ParseMode = tgbotapi.ModeHTML

	bot.HandledSend(resp)

	return nil
}
