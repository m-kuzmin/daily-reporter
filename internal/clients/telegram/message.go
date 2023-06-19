package telegram

import (
	"fmt"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/state"
	upd "github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
)

/*
telegramUpdateProcessor provides a uniform interface for processing telegram updates. The job of the implementor is to
call a correct method on `state` and return its result. This allows the caller to not know what the udate is, only know
that it knows how to apply itself to `state`.
*/
type updateProcessor interface {
	processTelegramUpdate(state state.Handler) (state.Handler, []response.BotAction)

	// Returns an ID that can be looked up in state storage. That state is then passed into `processTelegramUpdate`.
	//
	// Returns error if the update cannot be looked up in state storage
	stateHandle() (string, error)
}

type stateHandleError struct{}

func (stateHandleError) Error() string { return "unknown update, can't generate handle for it" }

/*
Represents a JSON response from the telegram API. Since an update could be many things like a message, button, poll
option, etc the update struct implements `UpdateProcessor` which performs the actions necessary to respond to an
update.
*/
type update struct {
	ID            int64                        `json:"update_id"`
	Message       option.Option[message]       `json:"message,omitempty"`
	CallbackQuery option.Option[callbackQuery] `json:"callback_query,omitempty"`
}

type message struct {
	ID             int                    `json:"message_id"`
	From           option.Option[user]    `json:"from,omitempty"`
	Chat           chat                   `json:"chat,omitempty"`
	ReplyToMessage option.Option[message] `json:"reply_to_message,omitempty"`
	Text           option.Option[string]  `json:"text,omitempty"`
}

type (
	callbackQuery struct{}
	user          struct {
		ID int64 `json:"id"`
	}
)

type chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// Identifies which type the message is and then calls a method on the state to handle it.
func (u *update) processTelegramUpdate(state state.Handler) (
	state.Handler, []response.BotAction,
) {
	if option.Some(true) == option.Flatmap(
		u.Message,
		func(message message) bool {
			return message.Chat.Type == string(upd.ChatTypePrivate)
		}) {
		return state.PrivateTextMessage(upd.PrivateTextMessage{
			Text: u.Message.MustUnwrap().Text.MustUnwrap(),
			Chat: upd.Chat{
				ID:   upd.ChatID(fmt.Sprint(u.Message.MustUnwrap().Chat.ID)),
				Type: upd.ChatType(u.Message.MustUnwrap().Chat.Type),
			},
		})
	}

	return state, []response.BotAction{}
}

func (u *update) stateHandle() (string, error) {
	if u.Message.IsSome() {
		message := u.Message.MustUnwrap()
		if message.From.IsNone() {
			return "", stateHandleError{}
		}

		switch message.Chat.Type {
		case string(upd.ChatTypePrivate):
			return fmt.Sprintf("private:%d", message.From.MustUnwrap().ID), nil
		default:
			return fmt.Sprintf("%d:%d", message.Chat.ID, message.From.MustUnwrap().ID), nil
		}
	}

	return "", stateHandleError{}
}
