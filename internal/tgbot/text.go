package tgbot

import (
	"fmt"
	"slices"
	"strings"

	"github.com/aybolid/wishbot/internal/db"
	"github.com/aybolid/wishbot/internal/logger"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func handleText(textMsg *tgbotapi.Message) error {
	logger.Sugared.Infow("handling text", "text", textMsg.Text, "chat_id", textMsg.Chat.ID, "from", textMsg.From)

	var err error

	if State.isPendingGroupCreation(textMsg.From.ID) {
		err = handleCreatingGroupFlow(textMsg)
	}
	if State.isPendingInviteCreation(textMsg.From.ID) {
		err = handleCreatingInviteFlow(textMsg)
	}
	if State.isPendingWishCreation(textMsg.From.ID) {
		err = handleCreatingWishFlow(textMsg)
	}

	return err
}

func handleCreatingGroupFlow(textMsg *tgbotapi.Message) error {
	group, err := db.CreateGroup(textMsg.From.ID, textMsg.Text)
	if err != nil {
		return err
	}

	State.releaseUser(textMsg.From.ID)

	resp := tgbotapi.NewMessage(textMsg.Chat.ID, fmt.Sprintf("Group \"%s\" was created!", group.Name))
	bot.HandledSend(resp)

	resp = tgbotapi.NewMessage(textMsg.Chat.ID, "Now you can add members to the group. /addmember")
	bot.HandledSend(resp)

	return nil
}

func handleCreatingInviteFlow(textMsg *tgbotapi.Message) error {
	groupID, ok := getPendingInviteCreation(textMsg.From.ID)
	if !ok {
		State.releaseUser(textMsg.From.ID)
		return fmt.Errorf("user is not pending invite creation")
	}

	mentions := getMentions(textMsg)

	if len(mentions) == 0 {
		resp := tgbotapi.NewMessage(textMsg.Chat.ID, "Please mention at least one user to invite.")
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
						textMsg.Chat.ID,
						fmt.Sprintf(
							"Seems like %s %s didn't chat with me yet. Please try again after they do.",
							mention.User.FirstName,
							mention.User.LastName,
						),
					)
					bot.HandledSend(resp)
					return
				}
			} else {
				// if it's a regular mention we need to extract the username
				// + 1 to skip the @ symbol
				userName := textMsg.Text[mention.Offset+1 : mention.Offset+mention.Length]
				logger.Sugared.Debugw("extracted user name from text", "username", userName)

				user, err = db.GetUserByUsername(userName)
				if err != nil {
					resp := tgbotapi.NewMessage(
						textMsg.Chat.ID,
						fmt.Sprintf("Seems like @%s didn't chat with me yet. Please try again after they do.", userName),
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
					textMsg.Chat.ID,
					fmt.Sprintf("@%s are already a member of the group.", user.Username),
				)
				bot.HandledSend(resp)
				return
			}

			// check if the user is trying to invite themself
			if user.UserID == textMsg.From.ID {
				logger.Sugared.Warnw("user tried to invite themself", "user_id", user.UserID)
				return
			}

			invite := groupInvite{
				invited: user,
				inviter: textMsg.From,
				groupId: groupID,
			}
			err = invite.sendInviteMessage()

			if err != nil {
				// notify the user if something went wrong
				resp := tgbotapi.NewMessage(
					textMsg.Chat.ID,
					fmt.Sprintf("Something went wrong while inviting %s. Please try again later.", user.Username),
				)
				bot.HandledSend(resp)
			} else {
				// notify the user if everything went fine
				logger.Sugared.Infow("invited user", "user_id", user.UserID, "chat_id", user.ChatID)
				resp := tgbotapi.NewMessage(textMsg.Chat.ID, fmt.Sprintf("Invited %s", user.Username))
				bot.HandledSend(resp)
			}
		}()
	}

	State.releaseUser(textMsg.From.ID)

	return nil
}

func handleCreatingWishFlow(textMsg *tgbotapi.Message) error {
	groupID, ok := getPendingWishCreation(textMsg.From.ID)
	if !ok {
		State.releaseUser(textMsg.From.ID)
		return fmt.Errorf("user is not pending wish creation")
	}

	wishURL := ""
	descriptionOffset := 0
	for _, entity := range textMsg.Entities {
		if entity.Type == "url" || entity.Type == "text_link" {
			if entity.Type == "text_link" {
				wishURL = entity.URL
			} else {
				wishURL = textMsg.Text[entity.Offset : entity.Offset+entity.Length]
			}
			descriptionOffset = entity.Offset + entity.Length
		}
	}

	if wishURL == "" {
		resp := tgbotapi.NewMessage(textMsg.Chat.ID, "No URL found! Send the URL and some description if applicable.")
		bot.HandledSend(resp)
		return nil
	}

	description := ""
	if len(textMsg.Text) > descriptionOffset {
		description = strings.TrimSpace(textMsg.Text[descriptionOffset:])
	}

	logger.Sugared.Debugw("creating wish", "wish_url", wishURL, "description", description)

	wish, err := db.CreateWish(wishURL, description, textMsg.From.ID, groupID)
	if err != nil {
		return err
	}

	group, err := db.GetGroup(groupID)
	if err != nil {
		resp := tgbotapi.NewMessage(textMsg.Chat.ID, "Something went wrong while trying to notify the group. Wish was created successfully though.")
		bot.HandledSend(resp)
		return nil
	}

	members, err := db.GetGroupMembers(groupID)
	if err != nil {
		resp := tgbotapi.NewMessage(textMsg.Chat.ID, "Something went wrong while trying to notify the group. Wish was created successfully though.")
		bot.HandledSend(resp)
		return nil
	}

	for _, member := range members {
		if member.UserID == textMsg.From.ID {
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
					"Hey! %s %s added a new wish to the \"%s\" group.",
					textMsg.From.FirstName,
					textMsg.From.LastName,
					group.Name,
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

	resp := tgbotapi.NewMessage(textMsg.Chat.ID, "Wish was created successfully!")
	bot.HandledSend(resp)

	State.releaseUser(textMsg.From.ID)
	return nil
}
