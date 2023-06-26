package response

import (
	"net/url"

	"github.com/m-kuzmin/daily-reporter/internal/util/option"
)

type BotAction interface {
	URLEncode() (endpont string, params url.Values)
}

// Noop returns empty list of bot actions.
func Noop() []BotAction { return []BotAction{} }

// NewSendMessage creates a new NewSendMessage and sets the default parse mode to "html".
func NewSendMessage(chatID ChatID, text string) SendMessage {
	return SendMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: option.Some("html"),
	}
}

type SendMessage struct {
	ChatID    ChatID
	Text      string
	ParseMode option.Option[string]
}

type ChatID string

// SetParseMode allows you to set the `ParseMode` and return `self` which allows for method chaining.
func (m SendMessage) SetParseMode(mode option.Option[string]) SendMessage {
	m.ParseMode = mode

	return m
}

func (m SendMessage) URLEncode() (string, url.Values) {
	var (
		endpoint = "sendMessage"
		params   = url.Values{}
	)

	params.Add("chat_id", string(m.ChatID))
	params.Add("text", m.Text)

	if parseMode, isSome := m.ParseMode.Unwrap(); isSome {
		params.Add("parse_mode", parseMode)
	}

	return endpoint, params
}
