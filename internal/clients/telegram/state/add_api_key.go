package state

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/github"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/util/logging"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
	"github.com/m-kuzmin/daily-reporter/internal/util/slashcmd"
)

type AddAPIKeyHandler struct {
	responses *addAPIKeyResponses
	userData  UserSharedData
	AddAPIKeyState
}

func (s *AddAPIKeyHandler) PrivateTextMessage(ctx context.Context, message update.PrivateTextMessage) Transition {
	cmd, isCmd := slashcmd.Parse(message.Text)
	if isCmd {
		switch strings.ToLower(cmd.Method) {
		case cancelCommand:
			logging.Debugf("%s %s Cancel /addApiKey ; Return to RootState", message.UpdateID.Log(), message.From.Log())

			return s.returnToRootStateWithMessage(message.Chat.ID, s.responses.Cancel)

		case noneCommand:
			s.userData.GithubAPIKey = option.None[string]()

			logging.Infof("%s API key deleted", message.From.Log())
			logging.Tracef("%s Return to RootState", message.UpdateID.Log())

			return s.returnToRootStateWithMessage(message.Chat.ID, s.responses.Deleted)
		}
	}

	client := github.NewClient(message.Text)

	login, err := client.Login(ctx)
	if err != nil {
		logging.Errorf("%s %s While saving GitHub API key: %s", message.UpdateID.Log(), message.From.Log(), err)

		return s.sameStateWithMessage(message.Chat.ID, s.responses.BadAPIKey)
	}

	s.userData.GithubAPIKey = option.Some(message.Text)

	logging.Infof("%s %s API key saved", message.UpdateID.Log(), message.From.Log())
	logging.Tracef("%s Return to RootState", message.UpdateID.Log())

	return NewTransition(s.RootState, s.userData, []response.BotAction{
		response.NewSendMessage(message.Chat.ID, fmt.Sprintf(s.responses.Success, login, login)).EnableWebPreview(),
	})
}

func (s *AddAPIKeyHandler) GroupTextMessage(_ context.Context, message update.GroupTextMessage) Transition {
	logging.Errorf("%s %s %s AddAPIKeyState should never be entered for any type of chat except private messages",
		message.UpdateID.Log(), message.Chat.Log(), message.From.Log())

	return s.returnToRootStateWithMessage(message.Chat.ID, s.responses.KeySentInPublicChat)
}

func (s *AddAPIKeyHandler) Ignore(_ context.Context) Transition {
	return NewTransition(s.AddAPIKeyState, s.userData, response.Nothing())
}

func (s *AddAPIKeyHandler) CallbackQuery(_ context.Context, cq update.CallbackQuery) Transition {
	logging.Infof("%s Ignoring callback query in AddApiKeyState", cq.Log())

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
