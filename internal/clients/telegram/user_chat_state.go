package telegram

import (
	"log"
	"net/url"
	"strconv"
	"strings"
)

func getConversationState(user user, chat chat) telegramConverser {
	return &rootConversationState{}
}
func setConversationState(user user, chat chat, new_state telegramConverser) {
	if getConversationState(user, chat) == new_state {
		return
	}
	panic("unimplemented")
}

type telegramConverser interface {
	telegramMessage(message) (telegramConverser, []telegramBotActor)
}

type telegramBotActor interface {
	telegramBotAction() (endpoint string, params url.Values)
}

type sendMessage struct {
	ChatID    string
	Text      string
	ParseMode string
}

func (m sendMessage) telegramBotAction() (endpoint string, params url.Values) {
	endpoint = "sendMessage"
	params = url.Values{}
	params.Add("chat_id", m.ChatID)
	params.Add("text", m.Text)
	if m.ParseMode != "" {
		params.Add("parse_mode", m.ParseMode)
	}

	return
}

type rootConversationState struct{}

func (s *rootConversationState) telegramMessage(message message) (telegramConverser, []telegramBotActor) {
	log.Printf("Got a message %q", *message.Text)
	if strings.TrimSpace(*message.Text) == "/start" {
		return s, []telegramBotActor{sendMessage{
			ChatID:    strconv.FormatInt(message.Chat.ID, 10),
			Text:      "Welcome\\!",
			ParseMode: "MarkdownV2",
		}}
	}
	return s, []telegramBotActor{}
}
