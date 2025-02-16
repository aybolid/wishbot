package tgbot

import (
	"fmt"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type cmdHandler func(cmdMsg *tgbotapi.Message) error

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

func handleCommand(cmdMsg *tgbotapi.Message) error {
	logger.Sugared.Infow("handling command", "command", cmdMsg.Text, "chat_id", cmdMsg.Chat.ID, "from", cmdMsg.From)
	State.releaseUser(cmdMsg.From.ID)

	var err error

	if handler, ok := cmdHandlers[cmdMsg.Text]; ok {
		err = handler(cmdMsg)
	} else {
		logger.Sugared.Errorw("unknown command received", "command", cmdMsg.Text, "chat_id", cmdMsg.Chat.ID, "from", cmdMsg.From)
	}

	return err
}

const MANAGE_MEMBERS_CALLBACK_PREFIX = "managemembers:"

func handleManageMembers(cmdMsg *tgbotapi.Message) error {
	groups, err := db.GetOwnedGroups(cmdMsg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "You don't have any owned groups yet. Please create one first. /creategroup")
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		if group.OwnerID != cmdMsg.From.ID {
			logger.Sugared.Errorw("not the owner of the group", "group_id", group.GroupID, "owner_id", group.OwnerID, "user_id", cmdMsg.From.ID)
			resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "You are not the owner of this group.")
			bot.HandledSend(resp)
			return nil
		}

		members, err := db.GetGroupMembers(group.GroupID)
		if err != nil {
			return err
		}

		filteredMembers := make([]*db.GroupMember, 0)
		for _, member := range members {
			if member.UserID == cmdMsg.From.ID {
				continue
			}
			filteredMembers = append(filteredMembers, member)
		}

		if len(filteredMembers) == 0 {
			resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "No members to manage found for this group.")
			bot.HandledSend(resp)
			return nil
		}

		resp := tgbotapi.NewMessage(
			cmdMsg.Chat.ID,
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
					cmdMsg.Chat.ID,
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
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "<b>Manage members.</b>\n\nSelect a group to manage members for.")

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

func handleManageWishes(cmdMsg *tgbotapi.Message) error {
	groups, err := db.GetUserGroups(cmdMsg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "You don't have any groups yet. Please create or join one first. /creategroup")
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
				cmdMsg.Chat.ID, fmt.Sprintf("No wishes found for the \"%s\" group. /addwish", group.Name),
			)
			bot.HandledSend(resp)
			return nil
		}

		resp := tgbotapi.NewMessage(
			cmdMsg.Chat.ID,
			fmt.Sprintf(
				"Here are your wishes from the \"%s\" group.",
				group.Name,
			),
		)
		bot.HandledSend(resp)

		for _, wish := range wishes {
			go func() {
				msg := tgbotapi.NewMessage(
					cmdMsg.Chat.ID,
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
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "<b>Manage wishes</b>\n\nSelect a group to manage wishes for.")

		resp.ReplyMarkup = getGroupSelectKeyboard(groups, func(group *db.Group) string {
			return fmt.Sprintf("%s%d", MANAGE_WISHES_CALLBACK_PREFIX, group.GroupID)
		})
		resp.ParseMode = tgbotapi.ModeHTML

		bot.HandledSend(resp)

		return nil
	}
}

const LEAVE_GROUP_CALLBACK_PREFIX = "leave_group:"

func handleLeaveGroup(cmdMsg *tgbotapi.Message) error {
	groups, err := db.GetUserGroups(cmdMsg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "You are not a member of any group.")
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		message := ""
		if group.OwnerID == cmdMsg.From.ID {
			message = fmt.Sprintf(
				"Do you really want to leave the \"%s\" group?\n<b>This action will delete the group, members and wishes as you are the owner.</b>",
				group.Name,
			)
		} else {
			message = fmt.Sprintf("Do you really want to leave the \"%s\" group?", group.Name)
		}

		sendAreYouSure(&areYouSureConfig{
			chatID:       cmdMsg.Chat.ID,
			message:      message,
			actionID:     LEAVE_GROUP_ACTION,
			callbackData: fmt.Sprintf("%d", group.GroupID),
		})

		return nil

	default:
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "<b>Leave group :(</b>\n\nSelect a group to leave.")

		resp.ReplyMarkup = getGroupSelectKeyboard(groups, func(group *db.Group) string {
			return fmt.Sprintf("%s%d", LEAVE_GROUP_CALLBACK_PREFIX, group.GroupID)
		})
		resp.ParseMode = tgbotapi.ModeHTML

		bot.HandledSend(resp)

		return nil
	}
}

func handleCancel(cmdMsg *tgbotapi.Message) error {
	// this command is meant to release the user from pending flows
	// as long as we do it in the command handler, we don't need to do anything here
	return nil
}

func handleStart(cmdMsg *tgbotapi.Message) error {
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
}

func handleCreateGroup(cmdMsg *tgbotapi.Message) error {
	userID := cmdMsg.From.ID
	State.setPendingGroupCreation(userID)

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

func handleAddMember(cmdMsg *tgbotapi.Message) error {
	groups, err := db.GetOwnedGroups(cmdMsg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "You don't have any created groups yet. Please create one first. /creategroup")
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		State.setPendingInviteCreation(cmdMsg.From.ID, group.GroupID)

		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, fmt.Sprintf("Please mention the users you want to invite to the \"%s\" group.", group.Name))
		bot.HandledSend(resp)

		return nil

	default:
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "<b>Invite another member.</b>\n\nSelect a group to add a member to (you can add members only to groups you created).")

		resp.ReplyMarkup = getGroupSelectKeyboard(groups, func(group *db.Group) string {
			return fmt.Sprintf("%s%d", INVITE_MEMBER_CALLBACK_PREFIX, group.GroupID)
		})
		resp.ParseMode = tgbotapi.ModeHTML

		bot.HandledSend(resp)

		return nil
	}
}

const ADD_WISH_CALLBACK_PREFIX = "add_wish:"

func handleAddWish(cmdMsg *tgbotapi.Message) error {
	groups, err := db.GetUserGroups(cmdMsg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "You don't have any groups yet. Please create or join one first.")
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]

		resp := tgbotapi.NewMessage(
			cmdMsg.Chat.ID,
			fmt.Sprintf(
				"Ok! Lets add a wish to the \"%s\" group.",
				group.Name,
			),
		)
		bot.HandledSend(resp)

		resp = tgbotapi.NewMessage(
			cmdMsg.Chat.ID,
			"Please send the URL of the wish you want to add with some description if applicable\\.\n\nExample:\n>>https://example\\.com\n>>This is a description",
		)
		resp.ParseMode = tgbotapi.ModeMarkdownV2
		bot.HandledSend(resp)

		State.setPendingWishCreation(cmdMsg.From.ID, group.GroupID)

		return nil

	default:
		resp := tgbotapi.NewMessage(
			cmdMsg.Chat.ID,
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

func handleWishes(cmdMsg *tgbotapi.Message) error {
	groups, err := db.GetUserGroups(cmdMsg.From.ID)
	if err != nil {
		return err
	}

	switch len(groups) {
	case 0:
		resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "You don't have any groups yet. Please create or join one first. /creategroup")
		bot.HandledSend(resp)
		return nil

	case 1:
		group := groups[0]
		wishes, err := db.GetGroupWishes(group.GroupID)
		if err != nil {
			return err
		}

		if len(wishes) == 0 {
			resp := tgbotapi.NewMessage(cmdMsg.Chat.ID, "No wishes found for this group. /addwish")
			bot.HandledSend(resp)
			return nil
		}

		var groupedByUser = make(map[int64][]*db.Wish)
		for _, wish := range wishes {
			groupedByUser[wish.UserID] = append(groupedByUser[wish.UserID], wish)
		}

		resp := tgbotapi.NewMessage(
			cmdMsg.Chat.ID,
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

				if user.UserID == cmdMsg.From.ID {
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
					cmdMsg.Chat.ID,
					text,
				)
				bot.HandledSend(resp)
			}()
		}

		return nil

	default:
		resp := tgbotapi.NewMessage(
			cmdMsg.Chat.ID,
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
