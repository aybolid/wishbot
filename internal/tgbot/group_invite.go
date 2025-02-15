package tgbot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const INVITE_MEMBER_CALLBACK_PREFIX = "invite_member:"
const ACCEPT_INVITE_CALLBACK_PREFIX = "accept_invite:"
const REJECT_INVITE_CALLBACK_PREFIX = "reject_invite:"

type groupInvite struct {
	invited *db.User
	inviter *tgbotapi.User
	groupID int64
}

func (i *groupInvite) sendInviteMessage() error {
	logger.Sugared.Infow("sending group invite message", "from", i.inviter.ID, "to", i.invited.UserID, "chat_id", i.invited.ChatID)

	group, err := db.GetGroup(i.groupID)
	if err != nil {
		return err
	}

	invite := tgbotapi.NewMessage(
		i.invited.ChatID,
		fmt.Sprintf(
			"<b>New group invite!</b>\n\nYou have been invited to join the \"<b>%s</b>\" group by <b>%s %s</b>.",
			group.Name,
			i.inviter.FirstName, i.inviter.LastName,
		),
	)

	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Reject", fmt.Sprintf("%s%d:%d", REJECT_INVITE_CALLBACK_PREFIX, i.inviter.ID, i.groupID)),
			tgbotapi.NewInlineKeyboardButtonData("Accept", fmt.Sprintf("%s%d:%d", ACCEPT_INVITE_CALLBACK_PREFIX, i.inviter.ID, i.groupID)),
		),
	)

	invite.ReplyMarkup = markup
	invite.ParseMode = tgbotapi.ModeHTML

	_, err = bot.Send(invite)

	if err != nil {
		logger.Sugared.Error(err)
	}
	return err
}

func parseInviteCallbackQuery(callbackQuery *tgbotapi.CallbackQuery, prefix string) (inviterID int64, groupID int64, err error) {
	if !strings.HasPrefix(callbackQuery.Data, prefix) {
		return 0, 0, fmt.Errorf("invalid invite callback query data")
	}

	parts := strings.Split(callbackQuery.Data[len(prefix):], ":")

	inviterID, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	groupID, err = strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return inviterID, groupID, nil
}
