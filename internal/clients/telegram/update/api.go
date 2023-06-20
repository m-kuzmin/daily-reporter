package update

import (
	"fmt"

	"github.com/m-kuzmin/daily-reporter/internal/util/option"
)

type Update struct {
	ID      UpdateID               `json:"update_id"`
	Message option.Option[Message] `json:"message,omitempty"`
}

//nolint:revive,golint // update.UpdateID is exactly what it should be named.
type UpdateID int64

/*
Returns `Some[string]` that refers to a conversation the user has with a bot. `None` if the conversation has no state.
*/
func (u Update) StateID() option.Option[string] {
	if message, isSome := u.Message.Unwrap(); isSome {
		if from, isSome := message.From.Unwrap(); isSome {
			return option.Some(fmt.Sprintf("%d:%d", message.Chat.ID, from.ID))
		}
	}

	return option.None[string]()
}

type Message struct {
	ID   MessageID             `json:"message_id"`
	From option.Option[User]   `json:"from"`
	Date int64                 `json:"date"`
	Chat Chat                  `json:"chat"`
	Text option.Option[string] `json:"text"`
}

type MessageID int64

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
