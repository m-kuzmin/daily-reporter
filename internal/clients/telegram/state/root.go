package state

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/github"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/template"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// Root is the default state
type Root struct {
	responses rootResponses
	userData  rootUserData
}

type rootUserData struct {
	GithubAPIKey option.Option[string]
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

const listProjectsCommand = "/listprojects"

func (s *Root) PrivateTextMessage(message update.PrivateTextMessage) (Handler, []response.BotAction) {
	switch strings.ToLower(strings.TrimSpace(message.Text)) {
	case "/start":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Start)}
	case "/help":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Help)}
	case "/addapikey":
		return &AddAPIKey{prevState: *s}, []response.BotAction{response.NewSendMessage(
			response.ChatID(fmt.Sprint(message.Chat.ID)), s.responses.AddAPIKey)}
	case listProjectsCommand:
		return s.handleListProjects(message.Chat.ID, option.None[string]())
	}

	// Parse /listProjects args
	command := strings.Split(message.Text, " ")

	const listProjectsAfterArgs = 3

	if len(command) == listProjectsAfterArgs {
		if strings.ToLower(command[0]) == listProjectsCommand {
			if command[1] == "after" {
				return s.handleListProjects(message.Chat.ID, option.Some(command[2]))
			}
		}
	}

	return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
		s.responses.UnknownMessage)}
}

//nolint:funlen
func (s *Root) handleListProjects(chatID update.ChatID, after option.Option[string]) (Handler, []response.BotAction) {
	const projectsOnPage = 10

	// Get the user's key
	key, isSome := s.userData.GithubAPIKey.Unwrap()
	if !isSome {
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)),
			s.responses.ListProjectsWithoutAPIKey)}
	}

	// Get the user's projects
	afterCursor := option.None[github.ProjectCursor]()

	if projectID := after.UnwrapOr(""); projectID != "" {
		afterCursor = option.Some(github.ProjectCursor(projectID))
	}

	projects, err := github.NewClient(key).ListViewerProjects(option.Some(projectsOnPage),
		afterCursor)
	if err != nil {
		log.Printf("While requesting user's projects: %s", err)

		var gqlerr *gqlerror.Error
		if errors.As(err, &gqlerr) {
			return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)),
				fmt.Sprintf("GitHub API error: %s", gqlerr.Message))}
		}

		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)),
			"Something went wrong while doing a GitHub API request")}
	}

	if len(projects) == 0 {
		if after.IsNone() {
			return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprintln(chatID)),
				s.responses.UserHasZeroProjects)}
		}

		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprintln(chatID)),
			s.responses.LastProjectsPage)}
	}

	// Print the projects
	projectList := "Your projects (10/page)"

	for _, project := range projects {
		projectList += fmt.Sprintf("\n\n<code>%s</code> <a href=%q><b>%s</b></a> (<a href=%q>%s</a>/%d)",
			project.Cursor,
			project.URL,
			project.Title,
			project.CreatorURL,
			project.CreatorLogin,
			project.Number,
		)
	}

	resp := response.NewSendMessage(response.ChatID(fmt.Sprintln(chatID)), projectList)

	resp = resp.SetReplyMarkup([][]response.InlineKeyboardButton{{
		response.InlineButtonSwitchQueryCurrentChat("Next page",
			"/listprojects after "+string(projects[len(projects)-1].Cursor)),
	}})

	return s, []response.BotAction{resp}
}

func (s *Root) GroupTextMessage(message update.GroupTextMessage) (Handler, []response.BotAction) {
	switch strings.ToLower(strings.TrimSpace(message.Text)) {
	case "/start":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Start)}
	case "/help":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Help)}
	case "/addapikey", listProjectsCommand:
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.PrivateCommandUsed)}
	default:
		return s, response.Nothing()
	}
}

func (s *Root) SetTemplate(template template.Template) error {
	group, err := template.Get("root")
	if err != nil {
		return fmt.Errorf(`while getting "root" group from template: %w`, err)
	}

	err = group.Populate(&s.responses)
	if err != nil {
		return fmt.Errorf(`while populating rootResponses from template: %w`, err)
	}

	return nil
}
