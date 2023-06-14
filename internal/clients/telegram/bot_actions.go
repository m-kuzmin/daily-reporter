package telegram

import (
	"net/url"
)

// Represents a generic action that can be sent back to telegram
type telegramBotActor interface {
	telegramBotAction() (endpoint string, params url.Values)
}

type sendMessage struct {
	ChatID    string
	Text      string
	ParseMode string
}

func (m sendMessage) telegramBotAction() (string, url.Values) {
	var (
		endpoint = "sendMessage"
		params   = url.Values{}
	)

	params.Add("chat_id", m.ChatID)
	params.Add("text", m.Text)

	if m.ParseMode != "" {
		params.Add("parse_mode", m.ParseMode)
	}

	return endpoint, params
}
