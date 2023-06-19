package response

import (
	"net/url"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
)

type BotAction interface {
	URLEncode() (endpont string, params url.Values)
}

// Returns empty list of bot actions
func Noop() []BotAction { return []BotAction{} }

func SendMessageBuilder(chatID update.ChatID, text string) SendMessage {
	return SendMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: option.None[string](),
	}
}

type SendMessage struct {
	ChatID    update.ChatID
	Text      string
	ParseMode option.Option[string]
}

func (m SendMessage) ParseModeHTML() SendMessage {
	m.ParseMode = option.Some("html")

	return m
}

func (m SendMessage) URLEncode() (string, url.Values) {
	var (
		endpoint = "sendMessage"
		params   = url.Values{}
	)

	params.Add("chat_id", string(m.ChatID))
	params.Add("text", m.Text)

	if m.ParseMode.IsSome() {
		params.Add("parse_mode", m.ParseMode.MustUnwrap())
	}

	return endpoint, params
}
