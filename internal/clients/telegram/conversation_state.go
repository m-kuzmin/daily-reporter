package telegram

import (
	"log"
	"strconv"
	"strings"
)

/*
Implementors of this interface are conversation states that can handle
different types of updates through methods. A method should return an
updated state of the convesation (if its the same then return self) and
a set of actions to perform on behalf of the bot.

E.g. the root state will handle all slash commands like /start, /help.
When there is a command that requires many steps from the user the
state has to change into that command's state. Once finished the state
usually returns back into the root state.

States can be struct{} (no fields) or hold internal state. This is helpful
if the command presents 2 buttons. The user presses one, does something,
then goes back into that 2 button state, but there's one left because they
have interacted with the first one.
*/
type ConversationStateHandler interface {
	telegramMessage(message) (ConversationStateHandler, []telegramBotActor)
}

// The default state
type rootConversationState struct{}

func (s *rootConversationState) telegramMessage(message message) (ConversationStateHandler, []telegramBotActor) {
	log.Printf("Got a message %q", *message.Text)

	switch strings.ToLower(strings.TrimSpace(*message.Text)) {
	case "/start":
		return s, []telegramBotActor{sendMessage{
			ChatID: strconv.FormatInt(message.Chat.ID, 10),
			Text: `Hi! I am a bot that can generate a report from your todo list on Github Projects.

You can use /help to get a list of commands. The one you will need right now is /addApiKey.`,
		}}

	case "/help":
		return s, []telegramBotActor{sendMessage{
			ChatID: strconv.FormatInt(message.Chat.ID, 10),
			Text: `Here are the commands I have:

/help: you are here!

/addApiKey: Add a GitHub API key`,
		}}

	case "/addapikey":
		return s, []telegramBotActor{sendMessage{
			ChatID: strconv.FormatInt(message.Chat.ID, 10),
			Text:   `501: Oops... We are working on it!`,
		}}

	default:
		log.Printf("Not handling %q", *message.Text)

		return s, []telegramBotActor{}
	}
}
