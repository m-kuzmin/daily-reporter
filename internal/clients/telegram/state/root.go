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

const listProjectsCommand = "listprojects"

// RootHandler is the default state
type RootHandler struct {
	responses *rootResponses
	userData  UserSharedData
}

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

	case "addapikey":
		return NewTransition(AddAPIKeyState{}, s.userData, []response.BotAction{
			response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)), s.responses.AddAPIKey),
		})

	case listProjectsCommand:
		after := option.None[github.ProjectCursor]()
		if afterV, hasAfter := cmd.NextAfter("after"); hasAfter && afterV != "" {
			after = option.Some(github.ProjectCursor(afterV))
		}

		return s.handleListProjects(message.Chat.ID, after)
	}

	return s.replyWithMessage(message.Chat.ID, s.responses.UnknownMessage)
}

func (s *RootHandler) handleListProjects(
	chatID update.ChatID, afterCursor option.Option[github.ProjectCursor],
) Transition {
	const projectsOnPage = 10

	// Get the user's key
	key, isSome := s.userData.GithubAPIKey.Unwrap()
	if !isSome {
		return s.replyWithMessage(chatID, s.responses.ListProjectsWithoutAPIKey)
	}

	// Get the user's projects
	projects, err := github.NewClient(key).ListViewerProjects(option.Some(projectsOnPage), afterCursor)
	if err != nil {
		log.Printf("While requesting user's projects: %s", err)

		if gqlMessage, is := github.GqlErrorString(err); is {
			return s.replyWithMessage(chatID, fmt.Sprintf("GitHub API error: %s", gqlMessage))
		}

		return s.replyWithMessage(chatID, "Something went wrong while doing a GitHub API request")
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
		projectList += fmt.Sprintf("\n\n<code>%s</code> <a href=%q><b>%s</b></a> (<a href=%q>%s</a>/%d)",
			project.Cursor, project.URL, project.Title, project.CreatorURL, project.CreatorLogin, project.Number)
	}

	projectListWithPagination := response.NewSendMessage(response.ChatID(fmt.Sprintln(chatID)), projectList)

	if len(projects) == projectsOnPage {
		projectListWithPagination = projectListWithPagination.SetReplyMarkup([][]response.InlineKeyboardButton{{
			response.InlineButtonSwitchQueryCurrentChat("Next page",
				fmt.Sprintf("/%s after %s", listProjectsCommand, projects[len(projects)-1].Cursor)),
		}})
	}

	return NewTransition(RootState{}, s.userData, []response.BotAction{projectListWithPagination})
}

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
	case "addapikey", listProjectsCommand:
		return s.replyWithMessage(message.Chat.ID, s.responses.PrivateCommandUsed)
	default:
		return s.Ignore()
	}
}

/*
replyWithMessage fills StateTranstion with values from self and sets the Response to a single message to the `chatID`
chat and `message` text.
*/
func (s RootHandler) replyWithMessage(chatID update.ChatID, message string) Transition {
	return NewTransition(RootState{}, s.userData,
		[]response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)), message)})
}

func (s *RootHandler) Ignore() Transition {
	return NewTransition(RootState{}, s.userData, response.Nothing())
}

type RootState struct{}

func (RootState) Handler(userData UserSharedData, responses *Responses) Handler {
	return &RootHandler{
		responses: &responses.Root,
		userData:  userData,
	}
}

type rootResponses struct {
	// commands

	Start     string `template:"start"`
	Help      string `template:"help"`
	AddAPIKey string `template:"addApiKey"`

	// warnings

	UserHasZeroProjects string `template:"userHasZeroProjects"`
	LastProjectsPage    string `template:"lastProjectsPage"`

	// errors

	PrivateCommandUsed        string `template:"privateCommandUsed"`
	UnknownMessage            string `template:"unknownMessage"`
	ListProjectsWithoutAPIKey string `template:"listProjectsWithoutApiKey"`
}

func newRootResponses(template template.Template) (rootResponses, error) {
	group, err := template.Get("root")
	if err != nil {
		return rootResponses{}, fmt.Errorf(`while getting "root" group from template: %w`, err)
	}

	resp := rootResponses{}

	err = group.Populate(&resp)
	if err != nil {
		return rootResponses{}, fmt.Errorf(`while populating rootResponses from template: %w`, err)
	}

	return resp, nil
}
