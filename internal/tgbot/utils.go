package tgbot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

// getMentions extracts all mentions from a message.
// text_mention and mention are both valid types.
func getMentions(msg *tgbotapi.Message) []tgbotapi.MessageEntity {
	var mentions []tgbotapi.MessageEntity
	for _, entity := range msg.Entities {
		if entity.Type == "text_mention" || entity.Type == "mention" {
			mentions = append(mentions, entity)
		}
	}
	return mentions
}
