package update

import "fmt"

type PrivateTextMessage struct {
	UpdateID UpdateID
	ID       MessageID
	Text     string
	Chat     Chat
	From     User
}

type GroupTextMessage struct {
	UpdateID UpdateID
	ID       MessageID
	Text     string
	Chat     Chat
	From     User
}

func (m PrivateTextMessage) Log() string {
	return fmt.Sprintf("(PrivateTextMessage %s %s %s %s (Text %q))",
		m.UpdateID.Log(),
		m.ID.Log(),
		m.Chat.Log(),
		m.From.Log(),
		m.Text,
	)
}

func (m GroupTextMessage) Log() string {
	return fmt.Sprintf("(GroupTextMessage %s %s %s %s (Text %q))",
		m.UpdateID.Log(),
		m.ID.Log(),
		m.Chat.Log(),
		m.From.Log(),
		m.Text,
	)
}
