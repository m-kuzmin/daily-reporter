package response

import (
	"net/url"

	"github.com/m-kuzmin/daily-reporter/internal/util/option"
)

type BotAction interface {
	URLEncode() (endpont string, params url.Values)
}

// Nothing returns an empty list of bot actions.
func Nothing() []BotAction { return []BotAction{} }

// NewSendMessage creates a new NewSendMessage and sets the default parse mode to "html".
func NewSendMessage(chatID ChatID, text string) SendMessage {
	return SendMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: option.Some("html"),
	}
}

type SendMessage struct {
	ChatID         ChatID
	Text           string
	ParseMode      option.Option[string]
	WebpagePreview bool
}

type ChatID string

// SetParseMode allows you to set the `ParseMode` and return `self` which allows for method chaining.
func (m SendMessage) SetParseMode(mode option.Option[string]) SendMessage {
	m.ParseMode = mode

	return m
}

// EnableWebPreview enables the preview that is visible below the message and displays the webpage content.
func (m SendMessage) EnableWebPreview() SendMessage {
	m.WebpagePreview = true

	return m
}

// EnableWebPreview enables the preview that is visible below the message and displays the webpage content.
func (m SendMessage) DisableWebPreview() SendMessage {
	m.WebpagePreview = false

	return m
}

func (m SendMessage) URLEncode() (string, url.Values) {
	var (
		endpoint = "sendMessage"
		params   = url.Values{}
	)

	params.Set("chat_id", string(m.ChatID))
	params.Set("text", m.Text)

	if parseMode, isSome := m.ParseMode.Unwrap(); isSome {
		params.Set("parse_mode", parseMode)
	}

	if !m.WebpagePreview {
		params.Set("disable_web_page_preview", "true")
	}

	return endpoint, params
}
