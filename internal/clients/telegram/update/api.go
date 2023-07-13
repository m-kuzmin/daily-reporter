package update

import (
	"fmt"

	"github.com/m-kuzmin/daily-reporter/internal/util/option"
)

type Update struct {
	ID            UpdateID                     `json:"update_id"`
	Message       option.Option[Message]       `json:"message,omitempty"`
	CallbackQuery option.Option[CallbackQuery] `json:"callback_query"`
}

//nolint:revive,golint // update.UpdateID is exactly what it should be named.
type UpdateID int64

/*
Returns `string, true` that refers to a conversation the user has with a bot. `"", false` if the conversation has no
state.
*/
func (u Update) StateID() (string, bool) {
	if message, isSome := u.Message.Unwrap(); isSome {
		if from, isSome := message.From.Unwrap(); isSome {
			return fmt.Sprintf("%d:%d", message.Chat.ID, from.ID), true
		}
	}

	if callback, isSome := u.CallbackQuery.Unwrap(); isSome {
		if message, isSome := callback.Message.Unwrap(); isSome {
			return fmt.Sprintf("%d:%d", message.Chat.ID, callback.From.ID), true
		}
	}

	return "", false
}

func (u Update) UserID() (UserID, bool) {
	if message, isSome := u.Message.Unwrap(); isSome {
		if from, isSome := message.From.Unwrap(); isSome {
			return from.ID, true
		}
	}

	return UserID(0), false
}

type Message struct {
	ID   MessageID             `json:"message_id"`
	From option.Option[User]   `json:"from"`
	Date int64                 `json:"date"`
	Chat Chat                  `json:"chat"`
	Text option.Option[string] `json:"text"`
}

type MessageID int64

type CallbackQuery struct {
	UpdateID UpdateID
	ID       CallbackQueryID        `json:"id"`
	From     User                   `json:"from"`
	Message  option.Option[Message] `json:"message,omitempty"`
	Data     option.Option[string]  `json:"data,omitempty"`
}

type CallbackQueryID string

type User struct {
	ID           UserID                `json:"id"`
	IsBot        bool                  `json:"is_bot"`
	FirstName    string                `json:"first_name"`
	LastName     option.Option[string] `json:"last_name"`
	Username     option.Option[string] `json:"username"`
	LanguageCode option.Option[string] `json:"language_code"`
}

type UserID int64

type Chat struct {
	ID   ChatID   `json:"id"`
	Type ChatType `json:"type"`
}

type ChatID int

type ChatType string

const (
	ChatTypePrivate    ChatType = "private"
	ChatTypeGroup      ChatType = "group"
	ChatTypeSuperGroup ChatType = "supergroup"
	ChatTypeChannel    ChatType = "channel"
)

func (u UpdateID) Log() string {
	return fmt.Sprintf("(UpdateID %d)", u)
}

func (m MessageID) Log() string {
	return fmt.Sprintf("(MessageID %d)", m)
}

func (u User) Log() string {
	return u.Username.Map(func(u string) string {
		return fmt.Sprintf("(User @%s)", u)
	}).UnwrapOr(fmt.Sprintf("(User %d)", u.ID))
}

func (c CallbackQuery) Log() string {
	return fmt.Sprintf("(CallbackQuery %s %s %s %s %s)",
		c.UpdateID.Log(),
		c.ID.Log(),
		c.From.Log(),
		option.Map[Message, string](c.Message, func(m Message) string {
			return m.Log()
		}).UnwrapOr("(Message nil)"),
		c.Data.Map(func(s string) string {
			return fmt.Sprintf("(Data %q)", s)
		}).UnwrapOr("(Data nil)"),
	)
}

func (c CallbackQueryID) Log() string {
	return fmt.Sprintf("(CallbackQueryID %q)", c)
}

func (m Message) Log() string {
	return fmt.Sprintf("(Message %s %s %s (Text %q))", m.ID.Log(),
		option.Map(m.From, func(m User) string { return m.Log() }).UnwrapOr("(From nil)"),
		m.Chat.Log(),
		m.Text,
	)
}

func (c Chat) Log() string {
	return fmt.Sprintf("(Chat (ChatType %s) (ChatID %d))", c.Type, c.ID)
}
