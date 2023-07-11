package telegram

type ZeroThreadsError struct{}

func (ZeroThreadsError) Error() string {
	return "telegram.Client.Start called with threads = 0, minimum = 1"
}
