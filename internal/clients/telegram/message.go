package telegram

import (
	"log"
)

/*
telegramUpdateProcessor provides a uniform interface for processing telegram updates.
The job of the struct implementing this interface is to call the correct method on
`state`. The state will do the actual work and the resulting state together with bot
actions is returned from the function.

Basically all it does is this:

	func(foo *Foo) processTelegramUpdate(state ConversationStateHander) (
		ConversationStateHandler,
		[]telegramBotActo
	) {
		return state.telegramMessage(foo)
	}
*/
type UpdateProcessor interface {
	processTelegramUpdate(state ConversationStateHandler) (ConversationStateHandler, []telegramBotActor)
}

/*
Represents a JSON response from the telegram API. Since an update could be many
things like a message, button, poll option, etc the update struct implements
`UpdateProcessor` which performs the actions necessary to respond to an update.
*/
type update struct {
	ID            int64          `json:"update_id"`
	Message       *message       `json:"message,omitempty"`
	CallbackQuery *callbackQuery `json:"callback_query,omitempty"`
}

type message struct {
	ID             int      `json:"message_id"`
	From           *user    `json:"from,omitempty"`
	Chat           *chat    `json:"chat,omitempty"`
	ReplyToMessage *message `json:"reply_to_message,omitempty"`
	Text           *string  `json:"text,omitempty"`
}

type callbackQuery struct{}

type user struct{}

type chat struct {
	ID int64 `json:"id"`
}

// Identifies which type the message is and then calls a method on the state to handle it.
func (u *update) processTelegramUpdate(state ConversationStateHandler) ( //nolint:ireturn
	ConversationStateHandler, []telegramBotActor,
) {
	switch {
	case u.Message != nil:
		if u.Message.From == nil || u.Message.Chat == nil || u.Message.Text == nil {
			return state, []telegramBotActor{}
		}

		return state.telegramMessage(*u.Message)
	default:
		log.Println("Not handling update", *u)

		return state, []telegramBotActor{}
	}
}
