package update

type PrivateTextMessage struct {
	Text string
	Chat Chat
}

type GroupTextMessage struct {
	Text string
	Chat Chat
}

type Chat struct {
	ID   ChatID
	Type ChatType
}

type ChatID string

type ChatType string

const (
	ChatTypePrivate    ChatType = "private"
	ChatTypeGroup      ChatType = "group"
	ChatTypeSuperGroup ChatType = "supergroup"
	ChatTypeChannel    ChatType = "channel"
)
