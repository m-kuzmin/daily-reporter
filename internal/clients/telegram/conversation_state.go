package telegram

import (
	"fmt"
	"log"
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/github"
)

/*
Implementors of this interface are conversation states that can handle different types of updates through methods. A
method should return an updated state of the convesation (if its the same then return self) and a set of actions to
perform on behalf of the bot.

E.g. the root state will handle all slash commands like /start, /help. When there is a command that requires many steps
from the user the state has to change into that command's state. Once finished the state usually returns back into the
root state.

States can be struct{} (no fields) or hold internal state. This is helpful if the command presents 2 buttons. The user
presses one, does something, then goes back into that 2 button state, but there's one left because they have interacted
with the first one.
*/
type ConversationStateHandler interface {
	telegramMessage(message) (ConversationStateHandler, []telegramBotActor)
}

// The default state
type rootConversationState struct{}

func (*rootConversationState) String() string {
	return "rootState"
}

func (s *rootConversationState) telegramMessage(message message) (ConversationStateHandler, []telegramBotActor) {
	if message.Chat.Type == chatTypePrivate {
		return s.privateMessage(message)
	} else {
		return s.publicChatMessage(message)
	}
}

const whoAmI = "I am a bot that can generate a report from your todo list on Github Projects."

func (s *rootConversationState) publicChatMessage(message message) (ConversationStateHandler, []telegramBotActor) {
	if message.Text == nil {
		return s, []telegramBotActor{}
	}
	switch strings.ToLower(strings.TrimSpace(*message.Text)) {
	case "/start":
		log.Printf("User %d used /start in %s", message.From.Id, message.Chat.Type)
		return s, []telegramBotActor{message.sameChatPlain("Hi! " + whoAmI + `

You can use /help to get a list of commands. To get started send me /addApiKey in private messages.`)}
	case "/help":
		log.Printf("User %d used /help in %s", message.From.Id, message.Chat.Type)
		return s, []telegramBotActor{message.sameChatPlain(s.helpText())}
	case "/addapikey", "/listprojects":
		log.Printf("User %d used /addApiKey in %s", message.From.Id, message.Chat.Type)
		return s, []telegramBotActor{message.sameChatPlain(`This command only works in private (direct) messages for your privacy and security.`)}
	default:
		return s, []telegramBotActor{}
	}
}

func (s *rootConversationState) privateMessage(message message) (ConversationStateHandler, []telegramBotActor) {
	if message.Text == nil {
		return s, []telegramBotActor{}
	}
	switch strings.ToLower(strings.TrimSpace(*message.Text)) {
	case "/start":
		log.Printf("User %d used /start in private messages", message.From.Id)
		return s, []telegramBotActor{message.sameChatPlain("Hi! " + whoAmI + `

You can use /help to get a list of commands. The one you will need right now is /addApiKey`)}
	case "/help":
		log.Printf("User %d used /help in private messages", message.From.Id)
		return s, []telegramBotActor{message.sameChatPlain(s.helpText())}
	case "/addapikey":
		log.Printf("User %d used /addApiKey in private messages", message.From.Id)
		return &addApiKeyConversationState{}, []telegramBotActor{message.sameChatMarkdownV2(
`Lets set your GitHub API key\. I can only hold one at a time and I will use it to get information about your projects\.

You can create a key on [this page](https://github.com/settings/tokens/new)\. *Make sure*:

• *You are the owner* of the account that you are adding the key for\.
• Only you and me \(this bot\) know the key because *its like a password*\.
• The key's permissions are _read:project_ and *that is it*\.

Once you have generated the key, send it here as a message\.
Be aware that once you close the key creation page you can no longer see it\. You yourself dont need to keep any copies, but if you fail to paste it in you'll have to delete the old one and generate a new one\.`)}
	default:
		return s, []telegramBotActor{message.sameChatPlain(`Sorry, I don't understand. Try /help maybe?`)}
	}
}

func (s *rootConversationState) helpText() string {
	return whoAmI + `
Here are the commands I have:

• /help: you are here!
• /addApiKey: Set your GitHub API key (Private messages only)`
}

type addApiKeyConversationState struct{}

func (s *addApiKeyConversationState) String() string {
	return "addApiKeyState"
}

func (s *addApiKeyConversationState) telegramMessage(message message) (ConversationStateHandler, []telegramBotActor) {
	if message.Chat.Type != chatTypePrivate {
		return &rootConversationState{}, []telegramBotActor{message.sameChatMarkdownV2(
			`If what you've sent me just now is a GitHub API token, *immediately* [revoke it](https://github.com/settings/tokens)\!
You should never try to add tokens in a public chat.`)}
	}

	if message.Text == nil {
		return s, []telegramBotActor{}
	}
	switch strings.TrimSpace(*message.Text) {
	case "/cancel":
		log.Printf("User %d canceled /addApiKey in %s", message.From.Id, message.Chat.Type)
		return &rootConversationState{}, []telegramBotActor{message.sameChatPlain("Canceled.")}
	default:
		client := github.NewClient(*message.Text)

		login, err := client.Login()
		if err != nil {
			log.Println("Error getting user's login by token:", err)
			return s, []telegramBotActor{message.sameChatPlain("Could not use your token, try again or /cancel")}
		}

		log.Printf("User %d successfully added their GitHub token", message.From.Id)
		return &rootConversationState{}, []telegramBotActor{message.sameChatHtml(
			fmt.Sprintf(`Nice to meet you, <a href="https://github.com/%s">%s</a>!`, login, login))}

	}
}
