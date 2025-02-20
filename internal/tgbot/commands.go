package tgbot

import (
	"fmt"
	"strings"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "You don't have any owned groups yet. Please create one first. /creategroup")
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		if group.OwnerID != ctx.msg.From.ID {
			logger.Sugared.Errorw("not the owner of the group", "group_id", group.GroupID, "owner_id", group.OwnerID, "user_id", ctx.msg.From.ID)
			resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, fmt.Sprintf("You are not the owner of the \"%s\" group.", group.Name))
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
			resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, fmt.Sprintf("No members to manage found for the \"%s\" group.", group.Name))
			bot.HandledSend(resp)
			return nil
		}

		resp := tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
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

				userWishes, err := db.GetUserWishes(member.UserID, group.GroupID)
				if err != nil {
					logger.Sugared.Errorw("failed to get user wishes for member display", "user_id", member.UserID, "err", err)
					return
				}

				msg := tgbotapi.NewMessage(
					ctx.msg.Chat.ID,
					fmt.Sprintf(
						"@%s\nThey have %d wishes.",
						user.Username,
						len(userWishes),
					),
				)

				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Kick", fmt.Sprintf("%s%d:%d", KICK_MEMBER_CALLBACK_PREFIX, member.UserID, group.GroupID)),
					),
				)

				bot.HandledSend(msg)
			}()
		}

		return nil

	default:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "<b>Manage members.</b>\n\nSelect a group to manage members for.")

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
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "You don't have any groups yet. Please create or join one first. /creategroup")
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
				ctx.msg.Chat.ID, fmt.Sprintf("No wishes found for the \"%s\" group. /addwish", group.Name),
			)
			bot.HandledSend(resp)
			return nil
		}

		resp := tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
			fmt.Sprintf(
				"Here are your wishes from the \"%s\" group.",
				group.Name,
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
						tgbotapi.NewInlineKeyboardButtonData("Delete", fmt.Sprintf("%s%d", DELETE_WISH_CALLBACK_PREFIX, wish.WishID)),
					),
				)

				bot.HandledSend(msg)
			}()
		}

		return nil

	default:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "<b>Manage wishes</b>\n\nSelect a group to manage wishes for.")

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
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "You are not a member of any group.")
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		message := ""
		if group.OwnerID == ctx.msg.From.ID {
			message = fmt.Sprintf(
				"Do you really want to leave the \"%s\" group?\n<b>This action will delete the group, members and wishes as you are the owner.</b>",
				group.Name,
			)
		} else {
			message = fmt.Sprintf("Do you really want to leave the \"%s\" group?", group.Name)
		}

		sendAreYouSure(&areYouSureConfig{
			chatID:       ctx.msg.Chat.ID,
			message:      message,
			actionID:     LEAVE_GROUP_ACTION,
			callbackData: fmt.Sprintf("%d", group.GroupID),
		})

		return nil

	default:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "<b>Leave group :(</b>\n\nSelect a group to leave.")

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
	resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, fmt.Sprintf("Hello, %s!", ctx.msg.From.FirstName))
	bot.HandledSend(resp)

	resp = tgbotapi.NewMessage(ctx.msg.Chat.ID, "I am a bot that will help you with sharing your wishes with your friends.")
	bot.HandledSend(resp)

	return nil
}

func handleCreateGroup(ctx *handleContext) error {
	userID := ctx.msg.From.ID
	State.setPendingGroupCreation(userID)

	resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "Please send the name for your new group")
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
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "You don't have any groups yet. Please create or join one first. /creategroup")
		bot.HandledSend(resp)
		return nil

	default:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "Here are your groups:")
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
						usernames[idx] += " (owner)"
					}
				}

				resp := tgbotapi.NewMessage(
					ctx.msg.Chat.ID,
					fmt.Sprintf(
						"<b>%s</b>\n\nThe group has %d members.\n%s",
						group.Name,
						len(users),
						strings.Join(usernames, ", "),
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
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "You don't have any created groups yet. Please create one first. /creategroup")
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		State.setPendingInviteCreation(ctx.msg.From.ID, group.GroupID)

		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, fmt.Sprintf("Please mention the users you want to invite to the \"%s\" group.", group.Name))
		bot.HandledSend(resp)

		return nil

	default:
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "<b>Invite another member.</b>\n\nSelect a group to add a member to (you can add members only to groups you created).")

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
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "You don't have any groups yet. Please create or join one first.")
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		resp := tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
			fmt.Sprintf(
				"Ok! Lets add a wish to the \"%s\" group.",
				group.Name,
			),
		)
		bot.HandledSend(resp)

		resp = tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
			"Please send the URL of the wish you want to add with some description if applicable\\.\n\nExample:\n>>https://example\\.com\n>>This is a description",
		)
		resp.ParseMode = tgbotapi.ModeMarkdownV2
		bot.HandledSend(resp)

		State.setPendingWishCreation(ctx.msg.From.ID, group.GroupID)

		return nil

	default:
		resp := tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
			"<b>Add new wish.</b>\n\nSelect a group to add a wish to. Created wish will be shared with all members of the group.",
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
		resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, "You don't have any groups yet. Please create or join one first. /creategroup")
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]
		wishes, err := db.GetGroupWishes(group.GroupID)
		if err != nil {
			return err
		}

		if len(wishes) == 0 {
			resp := tgbotapi.NewMessage(ctx.msg.Chat.ID, fmt.Sprintf("No wishes found for the \"%s\" group. /addwish", group.Name))
			bot.HandledSend(resp)
			return nil
		}

		var groupedByUser = make(map[int64][]*db.Wish)
		for _, wish := range wishes {
			groupedByUser[wish.UserID] = append(groupedByUser[wish.UserID], wish)
		}

		resp := tgbotapi.NewMessage(
			ctx.msg.Chat.ID,
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

				if user.UserID == ctx.msg.From.ID {
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
			"<b>View group wishes.</b>\n\nSelect a group and I will show you all wishes from that group.",
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
