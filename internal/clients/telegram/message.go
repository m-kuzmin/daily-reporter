package telegram

import (
	"log"
)

// telegramUpdateProcessor interface provides a uniform interface
// for processing telegram updates. An update only holds state about
// itself (an update) and then calls other functions to handle persistant
// state like conversation state or data about the user.
type UpdateProcessor interface {
	processTelegramUpdate() []telegramBotActor
}

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

func (u *update) processTelegramUpdate() []telegramBotActor {
	switch {
	case u.Message != nil:
		if u.Message.From == nil || u.Message.Chat == nil || u.Message.Text == nil {
			return []telegramBotActor{}
		}

		state := getConversationState(*u.Message.From, *u.Message.Chat)
		new_state, actions := state.telegramMessage(*u.Message)
		setConversationState(*u.Message.From, *u.Message.Chat, new_state)

		return actions
	default:
		log.Println("Not handling update", u)
		return []telegramBotActor{}
	}
}
