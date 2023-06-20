package update

type PrivateTextMessage struct {
	ID   MessageID
	Text string
	Chat Chat
	From User
}

type GroupTextMessage struct {
	ID   MessageID
	Text string
	Chat Chat
	From User
}
