package state

import (
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
)

type Handler interface {
	PrivateTextMessage(update.PrivateTextMessage) Transition
	GroupTextMessage(update.GroupTextMessage) Transition
	Ignore() Transition
}

type State interface {
	Handler(UserSharedData, *Responses) Handler
}

type UserSharedData struct {
	GithubAPIKey option.Option[string]
}

func NewUserSharedData() UserSharedData {
	return UserSharedData{
		GithubAPIKey: option.None[string](),
	}
}

// Transition represents changes to the state of the conversation as well as bot's responses to messages.
type Transition struct {
	// NewState is the state to use when processing the next message.
	NewState State
	// UserData is the user's shared data to use for the next message.
	UserData UserSharedData
	// Actions is the list of actions the bot should do in response to the current message
	Actions []response.BotAction
}

func NewTransition(
	newState State, userData UserSharedData, resp []response.BotAction,
) Transition {
	return Transition{
		NewState: newState,
		UserData: userData,
		Actions:  resp,
	}
}

func Handle(bot update.User, upd update.Update, state Handler) Transition {
	if message, isSome := upd.Message.Unwrap(); isSome {
		if transition, ok := handleMessage(bot, message, state); ok {
			return transition
		}
	}

	return state.Ignore()
}

func handleMessage(bot update.User, message update.Message, state Handler) (Transition, bool) {
	switch message.Chat.Type {
	case update.ChatTypePrivate:
		text, isSome := message.Text.Unwrap()
		if !isSome {
			return Transition{}, false
		}

		if username := bot.Username.UnwrapOr(""); username != "" {
			text = strings.TrimSpace(strings.TrimPrefix(text, "@"+username))
		}

		from, isSome := message.From.Unwrap()
		if !isSome {
			return Transition{}, false
		}

		return state.PrivateTextMessage(update.PrivateTextMessage{
			ID:   message.ID,
			Text: text,
			Chat: message.Chat,
			From: from,
		}), true
	case update.ChatTypeGroup:
		text, isSome := message.Text.Unwrap()
		if !isSome {
			return Transition{}, false
		}

		if username := bot.Username.UnwrapOr(""); username != "" {
			text = strings.TrimSpace(strings.TrimPrefix(text, "@"+username))
		}

		from, isSome := message.From.Unwrap()
		if !isSome {
			return Transition{}, false
		}

		return state.GroupTextMessage(update.GroupTextMessage{
			ID:   message.ID,
			Text: text,
			Chat: message.Chat,
			From: from,
		}), true
	case update.ChatTypeChannel, update.ChatTypeSuperGroup:
		return Transition{}, false
	}

	return Transition{}, false
}
