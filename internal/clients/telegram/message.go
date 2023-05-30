package telegram

// telegramUpdateProcessor interface provides a uniform interface
// for processing telegram updates. An update only holds state about
// itself (an update) and then calls other functions to handle persistant
// state like conversation state or data about the user.
type telegramUpdateProcessor interface {
	processTelegramUpdate()
}
