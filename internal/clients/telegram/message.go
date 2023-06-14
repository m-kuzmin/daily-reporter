package telegram

import (
	"log"
	"strconv"
)

/*
telegramUpdateProcessor provides a uniform interface for processing telegram updates. The job of the implementor is to
call a correct method on `state` and return its result. This allows the caller to not know what the udate is, only know
that it knows how to apply itself to `state`.
*/
type updateProcessor interface {
	processTelegramUpdate(state ConversationStateHandler) (ConversationStateHandler, []telegramBotActor)

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

// Generated a sendMessage with ChatID == message.Chat.ID
func (m *message) sameChatPlain(text string) sendMessage {
	return sendMessage{
		ChatID:    strconv.FormatInt(m.Chat.ID, 10),
		Text:      text,
		ParseMode: "",
	}
}

func (m *message) sameChatMarkdownV2(text string) sendMessage {
	return sendMessage{
		ChatID:    strconv.FormatInt(m.Chat.ID, 10),
		Text:      text,
		ParseMode: "MarkdownV2",
	}
}

func (m *message) sameChatHTML(text string) sendMessage {
	return sendMessage{
		ChatID:    strconv.FormatInt(m.Chat.ID, 10),
		Text:      text,
		ParseMode: "html",
	}
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

const (
	// Used in chat.Type and means that this is bot's direct messages
	chatTypePrivate = "private"
	chatTypeGroup   = "group"
)

// Identifies which type the message is and then calls a method on the state to handle it.
func (u *update) processTelegramUpdate(state ConversationStateHandler) (
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

func (u *update) stateHandle() (string, error) {
	switch {
	case u.Message != nil && u.Message.Chat.Type == chatTypePrivate && u.Message.From != nil:
		return "private:" + strconv.FormatInt(u.Message.From.ID, 10), nil
	case u.Message != nil && u.Message.Chat.Type == chatTypeGroup && u.Message.From != nil:
		return strconv.FormatInt(u.Message.Chat.ID, 10) + ":" + strconv.FormatInt(u.Message.From.ID, 10), nil
	default:
		return "", stateHandleError{}
	}
}
