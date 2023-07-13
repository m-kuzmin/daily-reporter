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

//nolint:cyclop,funlen // Unsplittable switch
func (s *RootHandler) PrivateTextMessage(ctx context.Context, message update.PrivateTextMessage) Transition {
	cmd, isCmd := slashcmd.Parse(message.Text)
	if !isCmd {
		logging.Tracef("%s Message ignored", message.Log())

		return s.replyWithMessage(message.Chat.ID, s.responses.UnknownMessage)
	}

	logging.Tracef("%s %s Used /%s", message.UpdateID.Log(), message.From.Log(), cmd.Method)

	switch strings.ToLower(cmd.Method) {
	case "start":
		return s.replyWithMessage(message.Chat.ID, s.responses.Start)

	case "help":
		return s.replyWithMessage(message.Chat.ID, s.responses.Help)

	case "dailystatus":
		if date, is := cmd.NextAfter("date"); is {
			logging.Tracef("%s %s /dailyStatus with date override", message.UpdateID.Log(), message.From.Log())

			return s.handleDailyStatus(ctx, message.UpdateID, message.From, message.Chat.ID, option.Some(date))
		}

		return s.handleDailyStatus(ctx, message.UpdateID, message.From, message.Chat.ID, option.None[string]())

	case "addapikey":
		if len(cmd.Args) == 1 {
			logging.Tracef("%s %s /addApiKey inline mode", message.UpdateID.Log(), message.From.Log())

			return s.handleAddAPIKeyInline(ctx, message.From, message.Chat.ID, cmd.Args[0])
		}

		logging.Debugf("%s %s Transition into AddApiKeyState", message.UpdateID.Log(), message.From.Log())

		return NewTransition(AddAPIKeyState{RootState: s.RootState}, s.userData, []response.BotAction{
			response.NewSendMessage(message.Chat.ID, s.responses.AddAPIKey),
		})

	case listProjectsCommand:
		after := option.None[github.ProjectCursor]()
		if afterV, hasAfter := cmd.NextAfter("after"); hasAfter && afterV != "" {
			after = option.Some(github.ProjectCursor(afterV))
		}

		return s.handleListProjects(ctx, message.From, message.Chat.ID, after)

	case "setdefaultproject":
		if len(cmd.Args) == 1 {
			return s.saveDefaultProject(ctx, cmd.Args[0], message.Chat.ID)
		}

		if s.userData.GithubAPIKey.IsSome() {
			logging.Tracef("%s Transition into SetDefaultProjectState", message.Log())

			return NewTransition(SetDefaultProjectState{RootState: s.RootState}, s.userData, []response.BotAction{
				response.NewSendMessage(message.Chat.ID, s.responses.SetDefaultProject),
			})
		}

		logging.Tracef("%s Tried to set default project without adding an API key", message.Log())

		return s.replyWithMessage(message.Chat.ID, s.responses.NoAPIKeyAdded)
	}

	logging.Tracef("%s Command ignored", message.Log())

	return s.replyWithMessage(message.Chat.ID, s.responses.UnknownMessage)
}

//nolint:cyclop // Unsplittable switch
func (s *RootHandler) GroupTextMessage(ctx context.Context, message update.GroupTextMessage) Transition {
	cmd, isCmd := slashcmd.Parse(message.Text)
	if !isCmd {
		logging.Tracef("%s Message ignored", message.Log())

		return s.Ignore(ctx)
	}

	logging.Tracef("%s %s %s Used /%s", message.UpdateID.Log(), message.Chat.Log(), message.From.Log(), cmd.Method)

	switch strings.ToLower(cmd.Method) {
	case "start":
		return s.replyWithMessage(message.Chat.ID, s.responses.Start)

	case "help":
		return s.replyWithMessage(message.Chat.ID, s.responses.Help)

	case "dailystatus":
		if date, is := cmd.NextAfter("date"); is {
			logging.Infof("%s /dailyStatus with date override", message.From.Log())

			return s.handleDailyStatus(ctx, message.UpdateID, message.From, message.Chat.ID, option.Some(date))
		}

		return s.handleDailyStatus(ctx, message.UpdateID, message.From, message.Chat.ID, option.None[string]())

	case "addapikey":
		if len(cmd.Args) != 0 {
			return s.replyWithMessage(message.Chat.ID, s.responses.APIKeySentInPublicChat)
		}

		return s.replyWithMessage(message.Chat.ID, s.responses.PrivateCommandUsed)

	case listProjectsCommand:
		return s.replyWithMessage(message.Chat.ID, s.responses.PrivateCommandUsed)

	case "setdefaultproject":
		if len(cmd.Args) == 1 {
			return s.saveDefaultProject(ctx, cmd.Args[0], message.Chat.ID)
		}

		if s.userData.GithubAPIKey.IsSome() {
			return NewTransition(SetDefaultProjectState{RootState: s.RootState}, s.userData, []response.BotAction{
				response.NewSendMessage(message.Chat.ID, s.responses.SetDefaultProject),
			})
		}

		return s.replyWithMessage(message.Chat.ID, s.responses.NoAPIKeyAdded)
	}

	logging.Tracef("%s Command ignored", message.Log())

	return s.Ignore(ctx)
}

func (s *RootHandler) CallbackQuery(_ context.Context, cq update.CallbackQuery) Transition {
	return NewTransition(s.RootState, s.userData, []response.BotAction{
		response.AnswerCallbackQuery{
			ID:        string(cq.ID),
			Text:      option.Some("This button doesnt work."),
			ShowAlert: false,
		},
	})
}

func (s *RootHandler) Ignore(_ context.Context) Transition {
	return NewTransition(s.RootState, s.userData, response.Nothing())
}

func (s *RootHandler) handleAddAPIKeyInline(ctx context.Context, user update.User, chatID update.ChatID, key string,
) Transition {
	client := github.NewClient(key)

	login, err := client.Login(ctx)
	if err != nil {
		logging.Errorf("While requesting user's GitHub username: %s", err)

		return s.replyWithMessage(chatID, s.responses.BadAPIKey)
	}

	s.userData.GithubAPIKey = option.Some(key)

	logging.Infof("%s Saved GitHub API Key", user.Log())

	return NewTransition(s.RootState, s.userData, []response.BotAction{
		response.NewSendMessage(chatID, fmt.Sprintf(s.responses.APIKeyAdded, login, login)).EnableWebPreview(),
	})
}

func (s *RootHandler) handleListProjects(
	ctx context.Context, user update.User, chatID update.ChatID, afterCursor option.Option[github.ProjectCursor],
) Transition {
	const projectsOnPage = 10

	// Get the user's key
	key, isSome := s.userData.GithubAPIKey.Unwrap()
	if !isSome {
		return s.replyWithMessage(chatID, s.responses.NoAPIKeyAdded)
	}

	// Get the user's projects
	projects, err := github.NewClient(key).ListViewerProjects(ctx, projectsOnPage, afterCursor)
	if err != nil {
		logging.Errorf("%s While getting projects for /listProjects %s", user.Log(), err)

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
		projectList += fmt.Sprintf(
			"\n\n<code>%s</code> <a href=%q><b>%s</b></a> (<a href=%q>%s</a>/%d)\nID: <code>%s</code>",
			project.Cursor, project.URL, project.Title,
			project.CreatorURL, project.CreatorLogin, project.Number,
			project.ID)
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

func (s *RootHandler) handleDailyStatus(ctx context.Context, updateID update.UpdateID, user update.User,
	chatID update.ChatID, dateOverride option.Option[string],
) Transition {
	key, isSome := s.userData.GithubAPIKey.Unwrap()

	if !isSome {
		logging.Debugf("%s %s /dailyStatus used without GitHub API key", updateID.Log(), user.Log())

		return NewTransition(s.RootState, s.userData, []response.BotAction{
			response.NewSendMessage(chatID, s.responses.NoAPIKeyAdded),
		})
	}

	const moreThanOne = 2

	projects, err := github.NewClient(key).ListViewerProjects(ctx, moreThanOne, option.None[github.ProjectCursor]())
	if err != nil {
		logging.Errorf("%s %s While collecting project list for /dailyStatus, GitHub error occurred: %s",
			updateID.Log(), user.Log(), err)

		return s.replyWithMessage(chatID,
			github.GqlErrorStringOr("GitHub API error: %s", err, s.responses.GithubErrorGeneric))
	}

	return s.maybeTransitionIntoDailyStatus(ctx, updateID, user, key, projects, chatID, dateOverride)
}

func (s *RootHandler) maybeTransitionIntoDailyStatus(ctx context.Context, updateID update.UpdateID, user update.User,
	apiKey string, projects []github.ProjectV2, chatID update.ChatID, dateOverride option.Option[string],
) Transition {
	switch len(projects) {
	case 0:
		logging.Debugf("%s %s Project list len is0 (according to genqlient), aborting /dailyStatus",
			updateID.Log(), user.Log())

		return NewTransition(s.RootState, s.userData, []response.BotAction{
			response.NewSendMessage(
				chatID, s.responses.UserHasZeroProjects,
			),
		})
	case 1:
		s.DefaultProject = option.Some(projects[0].ID)

		logging.Infof("%s Saved %q as the default project because the user only has 1 project", user.Log(), projects[0].Title)
		logging.Debugf("%s %s Transition into DailyStatusState", updateID.Log(), user.Log())

		return NewTransition(NewDailyStatusState(s.RootState, dateOverride), s.userData, []response.BotAction{
			response.NewSendMessage(chatID, fmt.Sprintf(s.responses.DailyStatus, projects[0].Title)),
		})
	default:
		projectID, isSome := s.DefaultProject.Unwrap()
		if !isSome {
			logging.Debugf("%s %s Aborting /dailyStatus because user has many projects, but no default is set",
				updateID.Log(), user.Log())

			return NewTransition(s.RootState, s.userData, []response.BotAction{
				response.NewSendMessage(chatID, s.responses.UseSetDefaultProject),
			})
		}

		defaultProject, err := github.NewClient(apiKey).ProjectV2ByID(ctx, projectID)
		if err != nil {
			logging.Errorf("%s While getting GitHub Project by ID for /dailyStatus: %s", user.Log(), err)

			return NewTransition(s.RootState, s.userData, []response.BotAction{
				response.NewSendMessage(chatID,
					github.GqlErrorStringOr("GitHub API error: %s", err, s.responses.GithubErrorGeneric)),
			})
		}

		logging.Debugf("%s %s Transition into DailyStatusState", updateID.Log(), user.Log())

		return NewTransition(NewDailyStatusState(s.RootState, dateOverride), s.userData, []response.BotAction{
			response.NewSendMessage(chatID, fmt.Sprintf(s.responses.DailyStatus, defaultProject.Title)),
		})
	}
}

func (s *RootHandler) saveDefaultProject(ctx context.Context, id string, chatID update.ChatID) Transition {
	token, isSome := s.userData.GithubAPIKey.Unwrap()
	if !isSome {
		return s.replyWithMessage(chatID, s.responses.NoAPIKeyAdded)
	}

	proj, err := github.NewClient(token).ProjectV2ByID(ctx, github.ProjectID(id))
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
	APIKeyAdded         string `template:"apiKeyAdded"`
	DailyStatus         string `template:"dailyStatus"`
	SavedDefaultProject string `template:"savedDefaultProject"`
	SetDefaultProject   string `template:"setDefaultProject"`

	// warnings

	UserHasZeroProjects  string `template:"userHasZeroProjects"`
	LastProjectsPage     string `template:"lastProjectsPage"`
	UseSetDefaultProject string `template:"useSetDefaultProject"`

	// errors

	PrivateCommandUsed     string `template:"privateCommandUsed"`
	UnknownMessage         string `template:"unknownMessage"`
	NoAPIKeyAdded          string `template:"noApiKeyAdded"`
	BadAPIKey              string `template:"badApiKey"`
	APIKeySentInPublicChat string `template:"apiKeySentInPublicChat"`
	GithubErrorGeneric     string `template:"githubErrorGeneric"`
}
