package state

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/github"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
	"github.com/m-kuzmin/daily-reporter/internal/util/slashcmd"
)

const (
	listProjectsCommand = "listprojects"
	noneCommand         = "none"
	cancelCommand       = "cancel"
)

// RootHandler is the default state
type RootHandler struct {
	responses *rootResponses
	userData  UserSharedData
	RootState
}

//nolint:cyclop
func (s *RootHandler) PrivateTextMessage(message update.PrivateTextMessage) Transition {
	cmd, isCmd := slashcmd.Parse(message.Text)
	if !isCmd {
		return s.replyWithMessage(message.Chat.ID, s.responses.UnknownMessage)
	}

	switch strings.ToLower(cmd.Method) {
	case "start":
		return s.replyWithMessage(message.Chat.ID, s.responses.Start)

	case "help":
		return s.replyWithMessage(message.Chat.ID, s.responses.Help)

	case "dailystatus":
		if date, is := cmd.NextAfter("date"); is {
			return s.handleDailyStatus(message.Chat.ID, option.Some(date))
		}

		return s.handleDailyStatus(message.Chat.ID, option.None[string]())

	case "addapikey":
		return NewTransition(AddAPIKeyState{RootState: s.RootState}, s.userData, []response.BotAction{
			response.NewSendMessage(message.Chat.ID, s.responses.AddAPIKey),
		})

	case listProjectsCommand:
		after := option.None[github.ProjectCursor]()
		if afterV, hasAfter := cmd.NextAfter("after"); hasAfter && afterV != "" {
			after = option.Some(github.ProjectCursor(afterV))
		}

		return s.handleListProjects(message.Chat.ID, after)

	case "setdefaultproject":
		if len(cmd.Args) == 1 {
			return s.saveDefaultProject(cmd.Args[0], message.Chat.ID)
		}

		if s.userData.GithubAPIKey.IsSome() {
			return NewTransition(SetDefaultProjectState{RootState: s.RootState}, s.userData, []response.BotAction{
				response.NewSendMessage(message.Chat.ID, s.responses.SetDefaultProject),
			})
		}

		return s.replyWithMessage(message.Chat.ID, s.responses.NoAPIKeyAdded)
	}

	return s.replyWithMessage(message.Chat.ID, s.responses.UnknownMessage)
}

//nolint:cyclop
func (s *RootHandler) GroupTextMessage(message update.GroupTextMessage) Transition {
	cmd, isCmd := slashcmd.Parse(message.Text)
	if !isCmd {
		return s.Ignore()
	}

	switch strings.ToLower(cmd.Method) {
	case "start":
		return s.replyWithMessage(message.Chat.ID, s.responses.Start)

	case "help":
		return s.replyWithMessage(message.Chat.ID, s.responses.Help)

	case "dailystatus":
		if date, is := cmd.NextAfter("date"); is {
			return s.handleDailyStatus(message.Chat.ID, option.Some(date))
		}

		return s.handleDailyStatus(message.Chat.ID, option.None[string]())

	case "addapikey", listProjectsCommand:
		return s.replyWithMessage(message.Chat.ID, s.responses.PrivateCommandUsed)

	case "setdefaultproject":
		if len(cmd.Args) == 1 {
			return s.saveDefaultProject(cmd.Args[0], message.Chat.ID)
		}

		if s.userData.GithubAPIKey.IsSome() {
			return NewTransition(SetDefaultProjectState{RootState: s.RootState}, s.userData, []response.BotAction{
				response.NewSendMessage(message.Chat.ID, s.responses.SetDefaultProject),
			})
		}

		return s.replyWithMessage(message.Chat.ID, s.responses.NoAPIKeyAdded)

	default:
		return s.Ignore()
	}
}

func (s *RootHandler) CallbackQuery(cq update.CallbackQuery) Transition {
	return NewTransition(s.RootState, s.userData, []response.BotAction{
		response.AnswerCallbackQuery{
			ID:        string(cq.ID),
			Text:      option.Some("This button doesnt work."),
			ShowAlert: false,
		},
	})
}

func (s *RootHandler) Ignore() Transition {
	return NewTransition(s.RootState, s.userData, response.Nothing())
}

func (s *RootHandler) handleListProjects(
	chatID update.ChatID, afterCursor option.Option[github.ProjectCursor],
) Transition {
	const projectsOnPage = 10

	// Get the user's key
	key, isSome := s.userData.GithubAPIKey.Unwrap()
	if !isSome {
		return s.replyWithMessage(chatID, s.responses.NoAPIKeyAdded)
	}

	// Get the user's projects
	projects, err := github.NewClient(key).ListViewerProjects(projectsOnPage, afterCursor)
	if err != nil {
		log.Printf("While requesting user's projects: %s", err)

		return s.replyWithMessage(chatID,
			github.GqlErrorStringOr("Github API error: %s", err, s.responses.GithubErrorGeneric))
	}

	if len(projects) == 0 {
		if afterCursor.IsNone() {
			return s.replyWithMessage(chatID, s.responses.UserHasZeroProjects)
		}

		return s.replyWithMessage(chatID, s.responses.LastProjectsPage)
	}

	// Print the projects
	projectList := fmt.Sprintf("Your projects (%d/page)", projectsOnPage)

	for _, project := range projects {
		projectList += fmt.Sprintf("\n\n<code>%s</code> <a href=%q><b>%s</b></a> (<a href=%q>%s</a>/%d)\nID: <code>%s</code>",
			project.Cursor, project.URL, project.Title, project.CreatorURL, project.CreatorLogin, project.Number, project.ID)
	}

	projectListWithPagination := response.NewSendMessage(chatID, projectList)

	if len(projects) == projectsOnPage {
		projectListWithPagination = projectListWithPagination.SetReplyMarkup([][]response.InlineKeyboardButton{{
			response.InlineButtonSwitchQueryCurrentChat("Next page",
				fmt.Sprintf("/%s after %s", listProjectsCommand, projects[len(projects)-1].Cursor)),
		}})
	}

	return NewTransition(s.RootState, s.userData, []response.BotAction{projectListWithPagination})
}

func (s *RootHandler) handleDailyStatus(chatID update.ChatID, dateOverride option.Option[string]) Transition {
	key, isSome := s.userData.GithubAPIKey.Unwrap()

	if !isSome {
		return NewTransition(s.RootState, s.userData, []response.BotAction{
			response.NewSendMessage(chatID, s.responses.NoAPIKeyAdded),
		})
	}

	const moreThanOne = 2

	projects, err := github.NewClient(key).ListViewerProjects(moreThanOne,
		option.None[github.ProjectCursor]())
	if err != nil {
		return s.replyWithMessage(chatID,
			github.GqlErrorStringOr("GitHub API error: %s", err, s.responses.GithubErrorGeneric))
	}

	return s.maybeTransitionIntoDailyStatus(context.Background(), key, projects, dateOverride, chatID)
}

func (s *RootHandler) maybeTransitionIntoDailyStatus(ctx context.Context, apiKey string, projects []github.ProjectV2,
	dateOverride option.Option[string], chatID update.ChatID,
) Transition {
	switch len(projects) {
	case 0:
		return NewTransition(s.RootState, s.userData, []response.BotAction{
			response.NewSendMessage(
				chatID, s.responses.UserHasZeroProjects,
			),
		})
	case 1:
		s.DefaultProject = option.Some(projects[0].ID)

		return NewTransition(NewDailyStatusState(s.RootState, dateOverride), s.userData, []response.BotAction{
			response.NewSendMessage(chatID, fmt.Sprintf(s.responses.DailyStatus, projects[0].Title)),
		})
	default:
		projectID, isSome := s.DefaultProject.Unwrap()
		if !isSome {
			return NewTransition(s.RootState, s.userData, []response.BotAction{
				response.NewSendMessage(
					chatID, s.responses.UseSetDefaultProject,
				),
			})
		}

		defaultProject, err := github.NewClient(apiKey).ProjectV2ByID(ctx, projectID)
		if err != nil {
			return NewTransition(s.RootState, s.userData, []response.BotAction{
				response.NewSendMessage(chatID,
					github.GqlErrorStringOr("GitHub API error: %s", err, s.responses.GithubErrorGeneric)),
			})
		}

		return NewTransition(NewDailyStatusState(s.RootState, dateOverride), s.userData, []response.BotAction{
			response.NewSendMessage(chatID, fmt.Sprintf(s.responses.DailyStatus, defaultProject.Title)),
		})
	}
}

func (s *RootHandler) saveDefaultProject(id string, chatID update.ChatID) Transition {
	token, isSome := s.userData.GithubAPIKey.Unwrap()
	if !isSome {
		return s.replyWithMessage(chatID, s.responses.NoAPIKeyAdded)
	}

	proj, err := github.NewClient(token).ProjectV2ByID(context.Background(), github.ProjectID(id))
	if err != nil {
		return s.replyWithMessage(chatID,
			github.GqlErrorStringOr("Github API error: %s", err, s.responses.GithubErrorGeneric))
	}

	s.DefaultProject = option.Some[github.ProjectID](github.ProjectID(id))

	return s.replyWithMessage(chatID, fmt.Sprintf("Saved %q as default project", proj.Title))
}

// replyWithMessage keeps the current state and user data but reponds with a single message into chat with text
func (s RootHandler) replyWithMessage(chatID update.ChatID, message string) Transition {
	return NewTransition(s.RootState, s.userData,
		[]response.BotAction{response.NewSendMessage(chatID, message)})
}

type RootState struct {
	DefaultProject option.Option[github.ProjectID]
}

func (s RootState) Handler(userData UserSharedData, responses *Responses) Handler {
	return &RootHandler{
		responses: &responses.Root,
		userData:  userData,
		RootState: s,
	}
}

type rootResponses struct {
	// command output

	Start               string `template:"start"`
	Help                string `template:"help"`
	AddAPIKey           string `template:"addApiKey"`
	DailyStatus         string `template:"dailyStatus"`
	SavedDefaultProject string `template:"savedDefaultProject"`
	SetDefaultProject   string `template:"setDefaultProject"`

	// warnings

	UserHasZeroProjects  string `template:"userHasZeroProjects"`
	LastProjectsPage     string `template:"lastProjectsPage"`
	UseSetDefaultProject string `template:"useSetDefaultProject"`

	// errors

	PrivateCommandUsed string `template:"privateCommandUsed"`
	UnknownMessage     string `template:"unknownMessage"`
	NoAPIKeyAdded      string `template:"noApiKeyAdded"`
	GithubErrorGeneric string `template:"githubErrorGeneric"`
}
