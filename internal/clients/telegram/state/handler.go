package state

import (
	"context"
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/util/logging"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
)

type Handler interface {
	PrivateTextMessage(context.Context, update.PrivateTextMessage) Transition
	GroupTextMessage(context.Context, update.GroupTextMessage) Transition
	CallbackQuery(context.Context, update.CallbackQuery) Transition
	// Ignore is called for all updates that a bot doesnt know how to process yet.
	Ignore(context.Context) Transition
	/*
		Unwind is called before the bot is shutdown and can be used to return a conversation to a "default" state. Use it to
		cancel or clean up any commands.
	*/
	// Unwind(context.Context) Transition
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

func Handle(ctx context.Context, bot update.User, upd update.Update, state Handler) Transition {
	if message, isSome := upd.Message.Unwrap(); isSome {
		if transition, ok := handleMessage(ctx, bot, message, upd.ID, state); ok {
			return transition
		}
	}

	if cq, isSome := upd.CallbackQuery.Unwrap(); isSome {
		return state.CallbackQuery(ctx, cq)
	}

	logging.Infof("%s Ignoring this update using state.Ignore()", upd.ID.Log())

	return state.Ignore(ctx)
}

func handleMessage(ctx context.Context, bot update.User, message update.Message, updateID update.UpdateID,
	state Handler,
) (Transition, bool) {
	switch message.Chat.Type {
	case update.ChatTypePrivate:
		text, isSome := message.Text.Unwrap()
		if !isSome {
			return Transition{}, false //nolint:exhaustruct // False indicates an error
		}

		if username := bot.Username.UnwrapOr(""); username != "" {
			text = strings.TrimSpace(strings.TrimPrefix(text, "@"+username))
		}

		from, isSome := message.From.Unwrap()
		if !isSome {
			return Transition{}, false //nolint:exhaustruct // False indicates an error
		}

		return state.PrivateTextMessage(ctx, update.PrivateTextMessage{
			UpdateID: updateID,
			ID:       message.ID,
			Text:     text,
			Chat:     message.Chat,
			From:     from,
		}), true
	case update.ChatTypeGroup:
		text, isSome := message.Text.Unwrap()
		if !isSome {
			return Transition{}, false //nolint:exhaustruct // False indicates an error
		}

		if username := bot.Username.UnwrapOr(""); username != "" {
			text = strings.TrimSpace(strings.TrimPrefix(text, "@"+username))
		}

		from, isSome := message.From.Unwrap()
		if !isSome {
			return Transition{}, false //nolint:exhaustruct // False indicates an error
		}

		return state.GroupTextMessage(ctx, update.GroupTextMessage{
			UpdateID: updateID,
			ID:       message.ID,
			Text:     text,
			Chat:     message.Chat,
			From:     from,
		}), true
	case update.ChatTypeChannel, update.ChatTypeSuperGroup:
		return Transition{}, false //nolint:exhaustruct // False indicates an error
	}

	return Transition{}, false //nolint:exhaustruct // False indicates an error
}
