package telegram

import (
	"fmt"
	"log"
	"strconv"
)

/*
telegramUpdateProcessor provides a uniform interface for processing telegram updates.
An implementation struct only holds fields about itself such as update id, message text, etc.
Conversation state is stored outside the update and an implementation can mutate it.

Caller of processTelegramUpdate doesn't know anything about what the update is. It doesn't
matter if its a message, poll option, etc. The update knows what it is and will do the things
it needs to.
*/
type UpdateProcessor interface {
	processTelegramUpdate(state ConversationStateHandler) (ConversationStateHandler, []telegramBotActor)
	stateHandle() (string, error)
}

/*
Represents a JSON response from the telegram API. Since an update could be many
things like a message, button, poll option, etc the update struct implements
`UpdateProcessor` which performs the actions neccessary to respond to an update.
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
func (m *message) sameChatHtml(text string) sendMessage {
	return sendMessage{
		ChatID:    strconv.FormatInt(m.Chat.ID, 10),
		Text:      text,
		ParseMode: "html",
	}
}

type callbackQuery struct{}
type user struct {
	Id int64 `json:"id"`
}
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
	ConversationStateHandler, []telegramBotActor) {
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
		return "private:" + strconv.FormatInt(u.Message.From.Id, 10), nil
	case u.Message != nil && u.Message.Chat.Type == chatTypeGroup && u.Message.From != nil:
		return strconv.FormatInt(u.Message.Chat.ID, 10) + ":" + strconv.FormatInt(u.Message.From.Id, 10), nil
	default:
		return "", fmt.Errorf("%s", fmt.Sprintf("Unknown update, cannot generate handle for it: %v", u))
	}
}
