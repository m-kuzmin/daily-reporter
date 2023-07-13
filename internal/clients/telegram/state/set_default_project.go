package state

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/github"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
	"github.com/m-kuzmin/daily-reporter/internal/util/slashcmd"
)

type SetDefaultProjectHandler struct {
	responses *SetDefaultProjectResponses
	userData  UserSharedData
	SetDefaultProjectState
}

func (s *SetDefaultProjectHandler) GroupTextMessage(ctx context.Context, message update.GroupTextMessage) Transition {
	return s.saveDefaultProject(ctx, message.Chat.ID, message.Text)
}

func (s *SetDefaultProjectHandler) PrivateTextMessage(ctx context.Context, message update.PrivateTextMessage,
) Transition {
	return s.saveDefaultProject(ctx, message.Chat.ID, message.Text)
}

func (s *SetDefaultProjectHandler) CallbackQuery(ctx context.Context, _ update.CallbackQuery) Transition {
	return s.Ignore(ctx)
}

func (s *SetDefaultProjectHandler) Ignore(_ context.Context) Transition {
	return NewTransition(s.SetDefaultProjectState, s.userData, response.Nothing())
}

func (s *SetDefaultProjectHandler) saveDefaultProject(ctx context.Context, chatID update.ChatID, text string,
) Transition {
	if cmd, is := slashcmd.Parse(text); is {
		switch strings.ToLower(cmd.Method) {
		case noneCommand:
			s.DefaultProject = option.None[github.ProjectID]()

			return NewTransition(s.RootState, s.userData, []response.BotAction{
				response.NewSendMessage(chatID, "Default project reset for this chat."),
			})
		case cancelCommand:
			return NewTransition(s.RootState, s.userData, []response.BotAction{
				response.NewSendMessage(chatID, "Canceled."),
			})
		}
	}

	token, isSome := s.userData.GithubAPIKey.Unwrap()
	if !isSome {
		return s.replyWithMessage(chatID, s.responses.NoAPIKeyAdded)
	}

	project, err := github.NewClient(token).ProjectV2ByID(ctx, github.ProjectID(text))
	if err != nil {
		return s.replyWithMessage(chatID,
			github.GqlErrorStringOr("Github API error: %s", err, s.responses.GithubErrorGeneric))
	}

	s.DefaultProject = option.Some[github.ProjectID](github.ProjectID(text))

	return NewTransition(s.RootState, s.userData, []response.BotAction{
		response.NewSendMessage(chatID, fmt.Sprintf(s.responses.Success, project.Title)),
	})
}

func (s SetDefaultProjectHandler) replyWithMessage(chatID update.ChatID, message string) Transition {
	return NewTransition(s.SetDefaultProjectState, s.userData, []response.BotAction{
		response.NewSendMessage(chatID, message),
	})
}

type SetDefaultProjectState struct {
	RootState
}

func (s SetDefaultProjectState) Handler(userData UserSharedData, resp *Responses) Handler {
	return &SetDefaultProjectHandler{
		responses:              &resp.SetDefaultProject,
		userData:               userData,
		SetDefaultProjectState: s,
	}
}

type SetDefaultProjectResponses struct {
	Success            string `template:"success"`
	GithubErrorGeneric string `template:"githubErrorGeneric"`
	NoAPIKeyAdded      string `template:"noApiKeyAdded"`
}
