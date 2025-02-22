package tgbot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/locals"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nicksnyder/go-i18n/v2/i18n"
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
	localizer    *i18n.Localizer
	chatID       int64
	message      string
	actionID     int
	callbackData string
}

type actionHandler = func(int, *handleContext) error

var actionHandlers = map[int]actionHandler{
	LEAVE_GROUP_ACTION: handleGroupLeave,
	DELETE_WISH_ACTION: handleDeleteWish,
	KICK_MEMBER_ACTION: handleKickMember,
}

func sendAreYouSure(config *areYouSureConfig) error {
	text := config.localizer.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "areYouSure",
		},
	)
	text += fmt.Sprintf("\n\n%s", config.message)
	msg := tgbotapi.NewMessage(config.chatID, text)

	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(config.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "no",
				},
			), ARE_YOU_SURE_NO_CALLBACK_PREFIX),
			tgbotapi.NewInlineKeyboardButtonData(
				config.localizer.MustLocalize(
					&i18n.LocalizeConfig{
						MessageID: "yes",
					},
				),
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

func handleNo(ctx *handleContext) error {
	return nil
}

func handleYes(ctx *handleContext) error {
	payload := strings.Split(ctx.callbackQuery.Data[len(ARE_YOU_SURE_YES_CALLBACK_PREFIX):], ":")

	logger.Sugared.Debugw("are you sure payload", "payload", payload)

	actionID, err := strconv.ParseInt(payload[0], 10, 64)
	if err != nil {
		return err
	}

	handler, ok := actionHandlers[int(actionID)]
	if ok {
		return handler(len(ARE_YOU_SURE_YES_CALLBACK_PREFIX)+len(payload[0])+1, ctx)
	} else {
		logger.Sugared.Errorw("no action handler for action id", "action_id", actionID)
	}

	return nil
}

func handleKickMember(dataOffset int, ctx *handleContext) error {
	payload := strings.Split(ctx.callbackQuery.Data[dataOffset:], ":")
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

	resp := tgbotapi.NewMessage(ctx.callbackQuery.Message.Chat.ID, ctx.localizer.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "youKickedMember",
			TemplateData: map[string]any{
				"Username":  user.Username,
				"GroupName": group.Name,
			},
		},
	))
	bot.HandledSend(resp)

	go func() {
		userLocalizer := locals.GetLocalizer(user.Language)

		msg := tgbotapi.NewMessage(
			user.ChatID,
			userLocalizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "youWereKickedNotification",
					TemplateData: map[string]any{
						"GroupName": group.Name,
					},
				},
			),
		)
		bot.HandledSend(msg)
	}()

	return nil
}

func handleDeleteWish(dataOffset int, ctx *handleContext) error {
	wishID, err := strconv.ParseInt(ctx.callbackQuery.Data[dataOffset:], 10, 64)
	if err != nil {
		return err
	}

	err = db.DeleteWish(wishID)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(ctx.callbackQuery.Message.Chat.ID, ctx.localizer.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "wishDeleted",
		},
	))
	bot.HandledSend(resp)

	return nil
}

func handleGroupLeave(dataOffset int, ctx *handleContext) error {
	groupID, err := strconv.ParseInt(ctx.callbackQuery.Data[dataOffset:], 10, 64)
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

	err = db.DeleteGroupMember(groupID, ctx.callbackQuery.From.ID)
	if err != nil {
		return err
	}

	resp := tgbotapi.NewMessage(ctx.callbackQuery.Message.Chat.ID, ctx.localizer.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "youLeftGroup",
			TemplateData: map[string]any{
				"GroupName": group.Name,
			},
		},
	))
	bot.HandledSend(resp)

	if group.OwnerID == ctx.callbackQuery.From.ID {
		resp = tgbotapi.NewMessage(
			ctx.callbackQuery.Message.Chat.ID,
			ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "groupDeleted",
				},
			),
		)
		bot.HandledSend(resp)

		for _, member := range members {
			if member.UserID == ctx.callbackQuery.From.ID {
				continue
			}
			go func() {
				user, err := db.GetUser(member.UserID)
				if err != nil {
					logger.Sugared.Errorw("error getting user for notification", "user_id", member.UserID, "error", err)
					return
				}

				userLocalizer := locals.GetLocalizer(user.Language)

				msg := tgbotapi.NewMessage(
					user.ChatID,
					userLocalizer.MustLocalize(
						&i18n.LocalizeConfig{
							MessageID: "groupDeletedNotification",
							TemplateData: map[string]any{
								"GroupName": group.Name,
							},
						},
					),
				)

				bot.HandledSend(msg)
			}()
		}
	} else {
		for _, member := range members {
			if member.UserID == ctx.callbackQuery.From.ID {
				continue
			}
			go func() {
				user, err := db.GetUser(member.UserID)
				if err != nil {
					logger.Sugared.Errorw("error getting user for notification", "user_id", member.UserID, "error", err)
					return
				}

				userLocalizer := locals.GetLocalizer(user.Language)

				msg := tgbotapi.NewMessage(
					user.ChatID,
					userLocalizer.MustLocalize(
						&i18n.LocalizeConfig{
							MessageID: "userLeftGroupNotification",
							TemplateData: map[string]any{
								"Username":  ctx.callbackQuery.From.FirstName,
								"GroupName": group.Name,
							},
						},
					),
				)

				bot.HandledSend(msg)
			}()
		}
	}

	return nil
}
