package state

import (
	"fmt"
	"log"
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/github"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
	"github.com/m-kuzmin/daily-reporter/internal/util/slashcmd"
)

type AddAPIKeyHandler struct {
	responses *addAPIKeyResponses
	userData  UserSharedData
	AddAPIKeyState
}

func (s *AddAPIKeyHandler) PrivateTextMessage(message update.PrivateTextMessage) Transition {
	cmd, isCmd := slashcmd.Parse(message.Text)
	if isCmd {
		switch strings.ToLower(cmd.Method) {
		case cancelCommand:
			return s.returnToRootStateWithMessage(message.Chat.ID, s.responses.Cancel)

		case noneCommand:
			s.userData.GithubAPIKey = option.None[string]()

			return s.returnToRootStateWithMessage(message.Chat.ID, s.responses.Deleted)
		}
	}

	client := github.NewClient(message.Text)

	login, err := client.Login()
	if err != nil {
		log.Printf("While requesting user's GitHub username: %s", err)

		return s.sameStateWithMessage(message.Chat.ID, s.responses.BadAPIKey)
	}

	s.userData.GithubAPIKey = option.Some(message.Text)

	return NewTransition(s.RootState, s.userData, []response.BotAction{
		response.NewSendMessage(message.Chat.ID, fmt.Sprintf(s.responses.Success, login, login)).EnableWebPreview(),
	})
}

func (s *AddAPIKeyHandler) GroupTextMessage(message update.GroupTextMessage) Transition {
	return s.returnToRootStateWithMessage(message.Chat.ID, s.responses.KeySentInPublicChat)
}

func (s *AddAPIKeyHandler) Ignore() Transition {
	return NewTransition(s.AddAPIKeyState, s.userData, response.Nothing())
}

func (s *AddAPIKeyHandler) CallbackQuery(cq update.CallbackQuery) Transition {
	return NewTransition(s.AddAPIKeyState, s.userData, []response.BotAction{
		response.AnswerCallbackQuery{
			ID:        string(cq.ID),
			Text:      option.Some("This button doesnt work."),
			ShowAlert: false,
		},
	})
}

/*
returnToRootStateWithMessage returns to RootState with current userdata and sends one message to `chatID` chat with
`message` text
*/
func (s AddAPIKeyHandler) returnToRootStateWithMessage(chatID update.ChatID, message string) Transition {
	return NewTransition(s.RootState, s.userData, []response.BotAction{
		response.NewSendMessage(chatID, message),
	})
}

/*
sameStateWithMessage keeps the current state with current userdata and sends one message to `chatID` chat with
`message` text
*/
func (s AddAPIKeyHandler) sameStateWithMessage(chatID update.ChatID, message string) Transition {
	return NewTransition(s.AddAPIKeyState, s.userData, []response.BotAction{
		response.NewSendMessage(chatID, message),
	})
}

type AddAPIKeyState struct {
	RootState
}

func (s AddAPIKeyState) Handler(userData UserSharedData, responses *Responses) Handler {
	return &AddAPIKeyHandler{
		responses:      &responses.AddAPIKey,
		userData:       userData,
		AddAPIKeyState: s,
	}
}

type addAPIKeyResponses struct {
	// Exit statuses

	Cancel  string `template:"cancel"`
	Success string `template:"success"`
	Deleted string `template:"deleted"`

	// Errors

	BadAPIKey           string `template:"badApiKey"`
	KeySentInPublicChat string `template:"keySentInPublicChat"`
	GithubErrorGeneric  string `template:"githubErrorGeneric"`
}
