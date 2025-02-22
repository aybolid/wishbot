package tgbot

import (
	"fmt"
	"strings"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type cmdHandler func(ctx *handleContext) error

var cmdHandlers = map[string]cmdHandler{
	"/start": handleStart,

	"/creategroup": handleCreateGroup,
	"/leavegroup":  handleLeaveGroup,
	"/mygroups":    handleMyGroups,

	"/addmember":     handleAddMember,
	"/managemembers": handleManageMembers,

	"/addwish":      handleAddWish,
	"/wishes":       handleWishes,
	"/managewishes": handleManageWishes,

	"/cancel": handleCancel,
}

func handleCommand(ctx *handleContext) error {
	logger.Sugared.Infow("handling command", "command", ctx.msg.Text, "chat_id", ctx.msg.Chat.ID, "from", ctx.msg.From)
	State.releaseUser(ctx.msg.From.ID)

	var err error

	if handler, ok := cmdHandlers[ctx.msg.Text]; ok {
		err = handler(ctx)
	} else {
		logger.Sugared.Errorw("unknown command received", "command", ctx.msg.Text, "chat_id", ctx.msg.Chat.ID, "from", ctx.msg.From)
	}

	return err
}

const MANAGE_MEMBERS_CALLBACK_PREFIX = "managemembers:"

func handleManageMembers(ctx *handleContext) error {
	groups, err := db.GetOwnedGroups(ctx.msg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "noOwnedGroups",
			},
		))
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		if group.OwnerID != ctx.msg.From.ID {
			logger.Sugared.Errorw("not the owner of the group", "group_id", group.GroupID, "owner_id", group.OwnerID, "user_id", ctx.msg.From.ID)
			resp := tgbotapi.NewMessage(
				ctx.msg.Chat.ID,
				ctx.localizer.MustLocalize(
					&i18n.LocalizeConfig{
						MessageID: "notOwner",
						TemplateData: map[string]string{
							"GroupName": group.Name,
						},
					},
				),
			)
			bot.HandledSend(resp)
			return nil
		}

		members, err := db.GetGroupMembers(group.GroupID)
		if err != nil {
			return err
		}

		filteredMembers := make([]*db.GroupMember, 0)
		for _, member := range members {
			if member.UserID == ctx.msg.From.ID {
				continue
			}
			filteredMembers = append(filteredMembers, member)
		}

		if len(filteredMembers) == 0 {
			resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "noMembers",
					TemplateData: map[string]string{
						"GroupName": group.Name,
					},
				},
			))
			bot.HandledSend(resp)
			return nil
		}

		resp := tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
			ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "hereAreMembers",
					TemplateData: map[string]string{
						"GroupName": group.Name,
					},
				},
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

				userWishes, err := db.GetUserWishes(member.UserID, group.GroupID)
				if err != nil {
					logger.Sugared.Errorw("failed to get user wishes for member display", "user_id", member.UserID, "err", err)
					return
				}

				msg := tgbotapi.NewMessage(
					ctx.msg.Chat.ID,
					ctx.localizer.MustLocalize(
						&i18n.LocalizeConfig{
							MessageID: "memberDisplay",
							TemplateData: map[string]any{
								"Username":  user.Username,
								"WishCount": len(userWishes),
							},
						},
					),
				)

				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(
							ctx.localizer.MustLocalize(
								&i18n.LocalizeConfig{
									MessageID: "kick",
								},
							),
							fmt.Sprintf("%s%d:%d", KICK_MEMBER_CALLBACK_PREFIX, member.UserID, group.GroupID),
						),
					),
				)

				bot.HandledSend(msg)
			}()
		}

		return nil

	default:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "manageMembers",
			},
		))

		resp.ReplyMarkup = getGroupSelectKeyboard(groups, func(group *db.Group) string {
			return fmt.Sprintf("%s%d", MANAGE_MEMBERS_CALLBACK_PREFIX, group.GroupID)
		})
		resp.ParseMode = tgbotapi.ModeHTML

		bot.HandledSend(resp)

		return nil
	}
}

const DELETE_WISH_CALLBACK_PREFIX = "delete_wish:"
const MANAGE_WISHES_CALLBACK_PREFIX = "manage_wishes:"

func handleManageWishes(ctx *handleContext) error {
	groups, err := db.GetUserGroups(ctx.msg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "noGroups",
			},
		))
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		wishes, err := db.GetGroupWishes(group.GroupID)
		if err != nil {
			return err
		}

		if len(wishes) == 0 {
			resp := tgbotapi.NewMessage(
				ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
					&i18n.LocalizeConfig{
						MessageID: "noWishesFound",
						TemplateData: map[string]string{
							"GroupName": group.Name,
						},
					},
				),
			)
			bot.HandledSend(resp)
			return nil
		}

		resp := tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
			ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "hereAreYourWishes",
					TemplateData: map[string]string{
						"GroupName": group.Name,
					},
				},
			),
		)
		bot.HandledSend(resp)

		for _, wish := range wishes {
			go func() {
				msg := tgbotapi.NewMessage(
					ctx.msg.Chat.ID,
					fmt.Sprintf(
						"%s\n%s\n\n",
						wish.URL,
						wish.Description,
					),
				)

				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData(ctx.localizer.MustLocalize(
							&i18n.LocalizeConfig{
								MessageID: "delete",
							},
						), fmt.Sprintf("%s%d", DELETE_WISH_CALLBACK_PREFIX, wish.WishID)),
					),
				)

				bot.HandledSend(msg)
			}()
		}

		return nil

	default:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "manageWishesMenu",
			},
		))

		resp.ReplyMarkup = getGroupSelectKeyboard(groups, func(group *db.Group) string {
			return fmt.Sprintf("%s%d", MANAGE_WISHES_CALLBACK_PREFIX, group.GroupID)
		})
		resp.ParseMode = tgbotapi.ModeHTML

		bot.HandledSend(resp)

		return nil
	}
}

const LEAVE_GROUP_CALLBACK_PREFIX = "leave_group:"

func handleLeaveGroup(ctx *handleContext) error {
	groups, err := db.GetUserGroups(ctx.msg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "noGroups",
			},
		))
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		message := ""
		if group.OwnerID == ctx.msg.From.ID {
			message = ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "leaveOwnedGroup",
					TemplateData: map[string]string{
						"GroupName": group.Name,
					},
				},
			)
		} else {
			message = ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "leaveGroup",
					TemplateData: map[string]string{
						"GroupName": group.Name,
					},
				},
			)
		}

		sendAreYouSure(&areYouSureConfig{
			localizer:    ctx.localizer,
			chatID:       ctx.msg.Chat.ID,
			message:      message,
			actionID:     LEAVE_GROUP_ACTION,
			callbackData: fmt.Sprintf("%d", group.GroupID),
		})

		return nil

	default:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "leaveGroupMenu",
			},
		))

		resp.ReplyMarkup = getGroupSelectKeyboard(groups, func(group *db.Group) string {
			return fmt.Sprintf("%s%d", LEAVE_GROUP_CALLBACK_PREFIX, group.GroupID)
		})
		resp.ParseMode = tgbotapi.ModeHTML

		bot.HandledSend(resp)

		return nil
	}
}

func handleCancel(ctx *handleContext) error {
	// this command is meant to release the user from pending flows
	// as long as we do it in the command handler, we don't need to do anything here
	return nil
}

func handleStart(ctx *handleContext) error {
	resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "hello",
			TemplateData: map[string]interface{}{
				"FirstName": ctx.msg.From.FirstName,
			},
		},
	))
	bot.HandledSend(resp)

	resp = tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "intro",
		},
	))
	bot.HandledSend(resp)

	return nil
}

func handleCreateGroup(ctx *handleContext) error {
	userID := ctx.msg.From.ID
	State.setPendingGroupCreation(userID)

	resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
		&i18n.LocalizeConfig{
			MessageID: "sendGroupName",
		},
	))
	bot.HandledSend(resp)
	return nil
}

func handleMyGroups(ctx *handleContext) error {
	userID := ctx.msg.From.ID
	groups, err := db.GetUserGroups(userID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "noGroups",
			},
		))
		bot.HandledSend(resp)
		return nil

	default:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "yourGroups",
			},
		))
		bot.HandledSend(resp)

		for _, group := range groups {
			go func() {
				members, err := db.GetGroupMembers(group.GroupID)
				if err != nil {
					logger.Sugared.Errorw("failed to get group members", "group_id", group.GroupID, "err", err)
					return
				}
				users := make([]*db.User, 0)
				for _, member := range members {
					user, err := db.GetUser(member.UserID)
					if err != nil {
						logger.Sugared.Errorw("failed to get user", "user_id", member.UserID, "err", err)
						return
					}
					users = append(users, user)
				}

				usernames := make([]string, len(users))
				for idx, user := range users {
					usernames[idx] = "@" + user.Username
					if user.UserID == group.OwnerID {
						usernames[idx] += " (⭐)"
					}
				}

				resp := tgbotapi.NewMessage(
					ctx.msg.Chat.ID,
					ctx.localizer.MustLocalize(
						&i18n.LocalizeConfig{
							MessageID: "groupEntry",
							TemplateData: map[string]any{
								"GroupName":   group.Name,
								"MemberCount": len(users),
								"Usernames":   strings.Join(usernames, ", "),
							},
						},
					),
				)

				resp.ParseMode = tgbotapi.ModeHTML
				bot.HandledSend(resp)
			}()
		}

		return nil
	}
}

func handleAddMember(ctx *handleContext) error {
	groups, err := db.GetOwnedGroups(ctx.msg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "noOwnedGroups",
			},
		))
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		State.setPendingInviteCreation(ctx.msg.From.ID, group.GroupID)

		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "mentionToInvite",
				TemplateData: map[string]any{
					"GroupName": group.Name,
				},
			},
		))
		bot.HandledSend(resp)

		return nil

	default:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "inviteMemberMenu",
			},
		))

		resp.ReplyMarkup = getGroupSelectKeyboard(groups, func(group *db.Group) string {
			return fmt.Sprintf("%s%d", INVITE_MEMBER_CALLBACK_PREFIX, group.GroupID)
		})
		resp.ParseMode = tgbotapi.ModeHTML

		bot.HandledSend(resp)

		return nil
	}
}

const ADD_WISH_CALLBACK_PREFIX = "add_wish:"

func handleAddWish(ctx *handleContext) error {
	groups, err := db.GetUserGroups(ctx.msg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "noGroups",
			},
		))
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		resp := tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
			ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "letsAddWish",
					TemplateData: map[string]any{
						"GroupName": group.Name,
					},
				},
			),
		)
		bot.HandledSend(resp)

		resp = tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
			ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "sendWishData",
				},
			),
		)
		resp.ParseMode = tgbotapi.ModeMarkdownV2
		bot.HandledSend(resp)

		State.setPendingWishCreation(ctx.msg.From.ID, group.GroupID)

		return nil

	default:
		resp := tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
			ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "addWishMenu",
				},
			),
		)

		resp.ReplyMarkup = getGroupSelectKeyboard(groups, func(group *db.Group) string {
			return fmt.Sprintf("%s%d", ADD_WISH_CALLBACK_PREFIX, group.GroupID)
		})
		resp.ParseMode = tgbotapi.ModeHTML

		bot.HandledSend(resp)

		return nil
	}
}

const DISPLAY_WISHES_CALLBACK_PREFIX = "display_wishes:"

func handleWishes(ctx *handleContext) error {
	groups, err := db.GetUserGroups(ctx.msg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
			&i18n.LocalizeConfig{
				MessageID: "noGroups",
			},
		))
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]
		wishes, err := db.GetGroupWishes(group.GroupID)
		if err != nil {
			return err
		}

		if len(wishes) == 0 {
			resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "noWishes",
					TemplateData: map[string]any{
						"GroupName": group.Name,
					},
				},
			))
			bot.HandledSend(resp)
			return nil
		}

		var groupedByUser = make(map[int64][]*db.Wish)
		for _, wish := range wishes {
			groupedByUser[wish.UserID] = append(groupedByUser[wish.UserID], wish)
		}

		resp := tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
			ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "hereAreWishes",
					TemplateData: map[string]string{
						"GroupName": group.Name,
					},
				},
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

				if user.UserID == ctx.msg.From.ID {
					text += ctx.localizer.MustLocalize(
						&i18n.LocalizeConfig{
							MessageID: "yourWishes",
						},
					)
				} else {
					text += ctx.localizer.MustLocalize(
						&i18n.LocalizeConfig{
							MessageID: "userWishes",
							TemplateData: map[string]interface{}{
								"Username": user.Username,
							},
						},
					)
				}
				text += "\n\n"

				for idx, wish := range wishes {
					text += fmt.Sprintf(
						"%d. %s\n%s\n\n",
						idx+1,
						wish.URL,
						wish.Description,
					)
				}

				resp := tgbotapi.NewMessage(
					ctx.msg.Chat.ID,
					text,
				)
				bot.HandledSend(resp)
			}()
		}

		return nil

	default:
		resp := tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
			ctx.localizer.MustLocalize(
				&i18n.LocalizeConfig{
					MessageID: "viewWishesMenu",
				},
			),
		)

		resp.ReplyMarkup = getGroupSelectKeyboard(groups, func(group *db.Group) string {
			return fmt.Sprintf("%s%d", DISPLAY_WISHES_CALLBACK_PREFIX, group.GroupID)
		})
		resp.ParseMode = tgbotapi.ModeHTML

		bot.HandledSend(resp)

		return nil
	}
}

func getGroupSelectKeyboard(groups []*db.Group, dataFn func(*db.Group) string) tgbotapi.InlineKeyboardMarkup {
	var buttons []tgbotapi.InlineKeyboardButton

	for _, group := range groups {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(group.Name, dataFn(group)))
	}

	return tgbotapi.NewInlineKeyboardMarkup(buttons)
}
