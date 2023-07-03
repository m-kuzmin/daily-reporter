package state

import (
	"fmt"
	"log"
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/github"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/template"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
	"github.com/m-kuzmin/daily-reporter/internal/util/slashcmd"
)

type AddAPIKeyHandler struct {
	responses *addAPIKeyResponses
	userData  UserSharedData
}

func (s *AddAPIKeyHandler) PrivateTextMessage(message update.PrivateTextMessage) Transition {
	cmd, isCmd := slashcmd.Parse(message.Text)
	if isCmd {
		switch strings.ToLower(cmd.Method) {
		case "cancel":
			return s.returnToRootStateWithMessage(message.Chat.ID, s.responses.Cancel)

		case "none":
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

	return NewTransition(RootState{}, s.userData, []response.BotAction{response.NewSendMessage(response.ChatID(
		fmt.Sprint(message.Chat.ID)), fmt.Sprintf(s.responses.Success, login, login)).EnableWebPreview()})
}

func (s *AddAPIKeyHandler) GroupTextMessage(message update.GroupTextMessage) Transition {
	return s.returnToRootStateWithMessage(message.Chat.ID, s.responses.KeySentInPublicChat)
}

func (s *AddAPIKeyHandler) Ignore() Transition {
	return NewTransition(AddAPIKeyState{}, s.userData, response.Nothing())
}

/*
returnToRootStateWithMessage returns to RootState with current userdata and sends one message to `chatID` chat with
`message` text
*/
func (s AddAPIKeyHandler) returnToRootStateWithMessage(chatID update.ChatID, message string) Transition {
	return NewTransition(RootState{}, s.userData, []response.BotAction{
		response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)), message),
	})
}

/*
sameStateWithMessage keeps the current state with current userdata and sends one message to `chatID` chat with
`message` text
*/
func (s AddAPIKeyHandler) sameStateWithMessage(chatID update.ChatID, message string) Transition {
	return NewTransition(AddAPIKeyState{}, s.userData, []response.BotAction{
		response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)), message),
	})
}

type AddAPIKeyState struct{}

func (AddAPIKeyState) Handler(userData UserSharedData, responses *Responses) Handler {
	return &AddAPIKeyHandler{
		responses: &responses.AddAPIKey,
		userData:  userData,
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
}

func newAddAPIKeyResponse(template template.Template) (addAPIKeyResponses, error) {
	group, err := template.Get("addApiKey")
	if err != nil {
		return addAPIKeyResponses{}, fmt.Errorf(`while getting "addApiKey" group from template: %w`, err)
	}

	resp := addAPIKeyResponses{}

	err = group.Populate(&resp)
	if err != nil {
		return addAPIKeyResponses{}, fmt.Errorf(`while populating addApiKeyResponses from template: %w`, err)
	}

	return resp, nil
}
