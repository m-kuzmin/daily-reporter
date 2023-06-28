package state

import (
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/template"
)

type Handler interface {
	PrivateTextMessage(update.PrivateTextMessage) (Handler, []response.BotAction)
	GroupTextMessage(update.GroupTextMessage) (Handler, []response.BotAction)
	SetTemplate(template.Template) error
}

func Handle(bot update.User, upd update.Update, state Handler) (Handler, []response.BotAction) {
	if message, isSome := upd.Message.Unwrap(); isSome {
		return handleMessage(bot, message, state)
	}

	return state, response.Nothing()
}

func handleMessage(bot update.User, message update.Message, state Handler) (Handler, []response.BotAction) {
	switch message.Chat.Type {
	case update.ChatTypePrivate:
		text, isSome := message.Text.Unwrap()
		if !isSome {
			return state, response.Nothing()
		}

		if username := bot.Username.UnwrapOr(""); username != "" {
			text = strings.TrimSpace(strings.TrimPrefix(text, "@"+username))
		}

		from, isSome := message.From.Unwrap()
		if !isSome {
			return state, response.Nothing()
		}

		return state.PrivateTextMessage(update.PrivateTextMessage{
			ID:   message.ID,
			Text: text,
			Chat: message.Chat,
			From: from,
		})
	case update.ChatTypeGroup:
		text, isSome := message.Text.Unwrap()
		if !isSome {
			return state, response.Nothing()
		}

		if username := bot.Username.UnwrapOr(""); username != "" {
			text = strings.TrimSpace(strings.TrimPrefix(text, "@"+username))
		}

		from, isSome := message.From.Unwrap()
		if !isSome {
			return state, response.Nothing()
		}

		return state.GroupTextMessage(update.GroupTextMessage{
			ID:   message.ID,
			Text: text,
			Chat: message.Chat,
			From: from,
		})
	case update.ChatTypeChannel, update.ChatTypeSuperGroup:
		return state, response.Nothing()
	}

	return state, response.Nothing()
}
