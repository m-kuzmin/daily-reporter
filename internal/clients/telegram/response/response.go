package response

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
	"github.com/pkg/errors"
)

type BotAction interface {
	JSONEncode() (endpoint string, _ json.RawMessage, _ error)
}

// Nothing returns an empty list of bot actions.
func Nothing() []BotAction { return []BotAction{} }

type SendMessage struct {
	ChatID                ChatID                `json:"chat_id"`
	Text                  string                `json:"text"`
	ParseMode             option.Option[string] `json:"parse_mode,omitempty"`
	DisableWebpagePreview bool                  `json:"disable_web_page_preview"`
	ReplyMarkup           ReplyMarkupper        `json:"reply_markup,omitempty"`
}

// NewSendMessage creates a new NewSendMessage and sets the default parse mode to "html" and disables web previews.
func NewSendMessage(chatID ChatID, text string) SendMessage {
	return SendMessage{
		ChatID:                chatID,
		Text:                  text,
		ParseMode:             option.Some("html"),
		DisableWebpagePreview: true,
		ReplyMarkup:           nil,
	}
}

func (m SendMessage) JSONEncode() (string, json.RawMessage, error) {
	body, err := json.Marshal(m)
	if err != nil {
		err = fmt.Errorf("while JSON encoding SendMessage: %w", err)
	}

	return "sendMessage", body, err
}

type ChatID string

// SetParseMode allows you to set the `ParseMode` and return `self` which allows for method chaining.
func (m SendMessage) SetParseMode(mode option.Option[string]) SendMessage {
	m.ParseMode = mode

	return m
}

// EnableWebPreview enables the preview that is visible below the message and displays the webpage content.
func (m SendMessage) EnableWebPreview() SendMessage {
	m.DisableWebpagePreview = false

	return m
}

// EnableWebPreview enables the preview that is visible below the message and displays the webpage content.
func (m SendMessage) DisableWebPreview() SendMessage {
	m.DisableWebpagePreview = true

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

	if !m.DisableWebpagePreview {
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
	CallbackData                 option.Option[string] `json:"callback_data"`
}

func InlineButtonSwitchQueryCurrentChat(text, query string) InlineKeyboardButton {
	return InlineKeyboardButton{ //nolint:exhauststruct // Other options should be None
		Text:                         text,
		SwitchInlineQueryCurrentChat: option.Some(query),
	}
}

// APIError from the telegram API.
type APIError struct {
	ErrorCode   int                `json:"error_code,omitempty"`
	Description string             `json:"description,omitempty"`
	Parameters  apiErrorParamaters `json:"parameters,omitempty"`
}

type apiErrorParamaters struct {
	MigrateToChatID option.Option[int64] `json:"migrate_to_chat_id,omitempty"`
	RertyAfter      option.Option[int]   `json:"retry_after,omitempty"`
}

func (e APIError) Error() string {
	return fmt.Sprintf("telegram API error: %d: %q", e.ErrorCode, e.Description)
}

type AnswerCallbackQuery struct {
	ID        string                `json:"callback_query_id"`
	Text      option.Option[string] `json:"text"`
	ShowAlert bool                  `json:"show_alert"`
}

func CallbackQueryAnswerNotification(cq update.CallbackQuery, text string) AnswerCallbackQuery {
	return AnswerCallbackQuery{
		ID:        string(cq.ID),
		Text:      option.Some(text),
		ShowAlert: false,
	}
}

func CallbackQueryAnswerAlert(id string, text string) AnswerCallbackQuery {
	return AnswerCallbackQuery{
		ID:        id,
		Text:      option.Some(text),
		ShowAlert: true,
	}
}

func (q AnswerCallbackQuery) JSONEncode() (string, json.RawMessage, error) {
	body, err := json.Marshal(q)
	if err != nil {
		err = fmt.Errorf("while JSON encoding AnswerCallbackQuery: %w", err)
	}

	return "answerCallbackQuery", body, err
}

type EditMessageReplyMarkup struct {
	ChatID      ChatID         `json:"chat_id"`
	MessageID   int64          `json:"message_id"`
	ReplyMarkup ReplyMarkupper `json:"reply_markup"`
}

func RemoveReplyMarkup(message update.Message) EditMessageReplyMarkup {
	return EditMessageReplyMarkup{
		ChatID:      ChatID(fmt.Sprint(message.Chat.ID)),
		MessageID:   int64(message.ID),
		ReplyMarkup: InlineKeyboardMarkup{Keyboard: [][]InlineKeyboardButton{{}}},
	}
}

func (m EditMessageReplyMarkup) JSONEncode() (string, json.RawMessage, error) {
	body, err := json.Marshal(m)
	if err != nil {
		err = fmt.Errorf("while JSON encoding EditMessageReplyMarkup: %w", err)
	}

	return "editMessageReplyMarkup", body, err
}
