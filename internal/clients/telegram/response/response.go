package response

import (
	"encoding/json"
	"log"
	"net/url"

	"github.com/m-kuzmin/daily-reporter/internal/util/option"
	"github.com/pkg/errors"
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
	ReplyMarkup    ReplyMarkupper
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

	if m.ReplyMarkup != nil {
		json, err := m.ReplyMarkup.ReplyMarkupJSON()
		if err != nil {
			log.Printf("While marshaling reply markup: %s", err)
		} else {
			params.Set("reply_markup", string(json))
		}
	}

	return endpoint, params
}

func (m SendMessage) SetReplyMarkup(markup [][]InlineKeyboardButton) SendMessage {
	m.ReplyMarkup = InlineKeyboardMarkup{Keyboard: markup}

	return m
}

type ReplyMarkupper interface {
	ReplyMarkupJSON() ([]byte, error)
}

type InlineKeyboardMarkup struct {
	Keyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

func (k InlineKeyboardMarkup) ReplyMarkupJSON() ([]byte, error) {
	marshaled, err := json.Marshal(k)

	return marshaled, errors.Wrap(err, "while marshaling InlineKeyboardMarkup to JSON")
}

/*
Only one `Option` should be `Some` and the doc comment on the option explains what it does. The text is always present
and is the button label
*/
type InlineKeyboardButton struct {
	// Button label
	Text string `json:"text"`

	// Pressing this button makes the user type "@Bot (string)" in the current chat, or just the bot's username.
	SwitchInlineQueryCurrentChat option.Option[string] `json:"switch_inline_query_current_chat"`
}

func InlineButtonSwitchQueryCurrentChat(text, query string) InlineKeyboardButton {
	return InlineKeyboardButton{
		Text:                         text,
		SwitchInlineQueryCurrentChat: option.Some(query),
	}
}
