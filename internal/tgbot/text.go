package tgbot

import (
	"fmt"
	"slices"
	"strings"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/locals"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func handleText(ctx *handleContext) error {
	logger.Sugared.Infow("handling text", "text", ctx.msg.Text, "chat_id", ctx.msg.Chat.ID, "from", ctx.msg.From)

	var err error

	if State.isPendingGroupCreation(ctx.msg.From.ID) {
		err = handleCreatingGroupFlow(ctx)
	}
	if State.isPendingInviteCreation(ctx.msg.From.ID) {
		err = handleCreatingInviteFlow(ctx)
	}
	if State.isPendingWishCreation(ctx.msg.From.ID) {
		err = handleCreatingWishFlow(ctx)
	}

	return err
}

func handleCreatingGroupFlow(ctx *handleContext) error {
	group, err := db.CreateGroup(ctx.msg.From.ID, ctx.msg.Text)
	if err != nil {
		return err
	}

	State.releaseUser(ctx.msg.From.ID)

	resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "groupCreatedNotification",
			TemplateData: map[string]any{
				"GroupName": group.Name,
			},
		},
	))
	bot.HandledSend(resp)

	resp = tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "canAddMembers",
		},
	))
	bot.HandledSend(resp)

	return nil
}

func handleCreatingInviteFlow(ctx *handleContext) error {
	groupID, ok := getPendingInviteCreation(ctx.msg.From.ID)
	if !ok {
		State.releaseUser(ctx.msg.From.ID)
		return fmt.Errorf("user is not pending invite creation")
	}

	mentions := getMentions(ctx.msg)

	if len(mentions) == 0 {
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "mentionOneUser",
			},
		))
		bot.HandledSend(resp)
		return nil
	}

	groupMembers, err := db.GetGroupMembers(groupID)
	if err != nil {
		return err
	}

	for _, mention := range mentions {
		go func() {
			var err error
			var user *db.User

			if mention.User != nil {
				// if it's a text_mention we can use the user object
				user, err = db.GetUser(mention.User.ID)
				if err != nil {
					resp := tgbotapi.NewMessage(
						ctx.msg.Chat.ID,
						ctx.localizer.MustLocalize(
							&i18n.LocalizeConfig{
								MessageID: "didntChatWithUser",
								TemplateData: map[string]any{
									"Username": mention.User.FirstName,
								},
							},
						),
					)
					bot.HandledSend(resp)
					return
				}
			} else {
				// if it's a regular mention we need to extract the username
				// + 1 to skip the @ symbol
				userName := ctx.msg.Text[mention.Offset+1 : mention.Offset+mention.Length]
				logger.Sugared.Debugw("extracted user name from text", "username", userName)

				user, err = db.GetUserByUsername(userName)
				if err != nil {
					resp := tgbotapi.NewMessage(
						ctx.msg.Chat.ID,
						ctx.localizer.MustLocalize(
							&i18n.LocalizeConfig{
								MessageID: "didntChatWithUser",
								TemplateData: map[string]any{
									"Username": "@" + userName,
								},
							},
						),
					)
					bot.HandledSend(resp)
					return
				}
			}

			// check if the user is already a member of the group
			if slices.ContainsFunc(groupMembers, func(m *db.GroupMember) bool {
				return m.UserID == user.UserID
			}) {
				resp := tgbotapi.NewMessage(
					ctx.msg.Chat.ID,
					ctx.localizer.MustLocalize(
						&i18n.LocalizeConfig{
							MessageID: "alreadyAMember",
							TemplateData: map[string]any{
								"Username": user.Username,
							},
						},
					),
				)
				bot.HandledSend(resp)
				return
			}

			// check if the user is trying to invite themself
			if user.UserID == ctx.msg.From.ID {
				logger.Sugared.Warnw("user tried to invite themself", "user_id", user.UserID)
				return
			}

			invite := groupInvite{
				invited: user,
				inviter: ctx.msg.From,
				groupID: groupID,
			}
			err = invite.sendInviteMessage()

			if err != nil {
				// notify the user if something went wrong
				resp := tgbotapi.NewMessage(
					ctx.msg.Chat.ID,
					ctx.localizer.MustLocalize(
						&i18n.LocalizeConfig{
							MessageID: "errorInvitingUser",
							TemplateData: map[string]any{
								"Username": "@" + user.Username,
							},
						},
					),
				)
				bot.HandledSend(resp)
			} else {
				// notify the user if everything went fine
				logger.Sugared.Infow("invited user", "user_id", user.UserID, "chat_id", user.ChatID)
				resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, fmt.Sprintf("Invited %s", user.Username))
				bot.HandledSend(resp)
			}
		}()
	}

	State.releaseUser(ctx.msg.From.ID)

	return nil
}

func handleCreatingWishFlow(ctx *handleContext) error {
	groupID, ok := getPendingWishCreation(ctx.msg.From.ID)
	if !ok {
		State.releaseUser(ctx.msg.From.ID)
		return fmt.Errorf("user is not pending wish creation")
	}

	wishURL := ""
	descriptionOffset := 0
	for _, entity := range ctx.msg.Entities {
		if entity.Type == "url" || entity.Type == "text_link" {
			if entity.Type == "text_link" {
				wishURL = entity.URL
			} else {
				wishURL = ctx.msg.Text[entity.Offset : entity.Offset+entity.Length]
			}
			descriptionOffset = entity.Offset + entity.Length
		}
	}

	if wishURL == "" {
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "errorNoURL",
			},
		))
		bot.HandledSend(resp)
		return nil
	}

	description := ""
	if len(ctx.msg.Text) > descriptionOffset {
		description = strings.TrimSpace(ctx.msg.Text[descriptionOffset:])
	}

	logger.Sugared.Debugw("creating wish", "wish_url", wishURL, "description", description)

	wish, err := db.CreateWish(wishURL, description, ctx.msg.From.ID, groupID)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "errorWishGroupNotification",
			},
		))
		bot.HandledSend(resp)
		return nil
	}

	members, err := db.GetGroupMembers(groupID)
	if err != nil {
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "errorWishGroupNotification",
			},
		))
		bot.HandledSend(resp)
		return nil
	}

	for _, member := range members {
		if member.UserID == ctx.msg.From.ID {
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
						MessageID: "wishCreatedGroupNotification",
						TemplateData: map[string]any{
							"Username":  ctx.msg.From.FirstName,
							"GroupName": group.Name,
						},
					},
				),
			)
			bot.HandledSend(msg)

			wishMsg := tgbotapi.NewMessage(
				user.ChatID,
				fmt.Sprintf(
					"%s\n\n%s",
					wish.URL,
					wish.Description,
				),
			)
			bot.HandledSend(wishMsg)
		}()
	}

	resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "wishCreatedNotification",
		},
	))
	bot.HandledSend(resp)

	State.releaseUser(ctx.msg.From.ID)
	return nil
}
