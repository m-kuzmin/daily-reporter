package state

import (
	"fmt"
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/github"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/template"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
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

	// errors

	PrivateCommandUsed        string `template:"privateCommandUsed"`
	UnknownMessage            string `template:"unknownMessage"`
	ListProjectsWithoutAPIKey string `template:"listProjectsWithoutApiKey"`
	GitHubAPIErrorGeneric     string `template:"githubApiErrorGeneric"`
}

func (s *Root) PrivateTextMessage(message update.PrivateTextMessage) (Handler, []response.BotAction) {
	switch strings.ToLower(strings.TrimSpace(message.Text)) {
	case "/start":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Start)}
	case "/help":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Help)}
	case "/addapikey":
		return &AddAPIKey{}, []response.BotAction{response.NewSendMessage(
			response.ChatID(fmt.Sprint(message.Chat.ID)), s.responses.AddAPIKey)}
	case "/listprojects":
		return s.handleListProjects(message)
	default:
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.UnknownMessage)}
	}
}

func (s *Root) handleListProjects(message update.PrivateTextMessage) (Handler, []response.BotAction) {
	// Get the user's key
	key, isSome := s.userData.GithubAPIKey.Unwrap()
	if !isSome {
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.ListProjectsWithoutAPIKey)}
	}

	// Get the user's projects
	projects, err := github.NewClient(key).ListViewerProjects(option.None[int](),
		option.None[github.ProjectCursor]())
	if err != nil {
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.GitHubAPIErrorGeneric)}
	}

	if len(projects) == 0 {
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprintln(message.Chat.ID)),
			s.responses.UserHasZeroProjects)}
	}

	// Print the projects
	projectList := "Your projects (page 1)"

	for _, project := range projects {
		id := project.ID //nolint:varnamelen

		const truncateIDLen = 7
		if len(id) > truncateIDLen {
			id = id[len(id)-truncateIDLen:]
		}

		projectList += fmt.Sprintf("\n\n%s: <a href=%q><b>%s</b></a> (<a href=%q>%s</a>/%d)",
			id,
			project.URL,
			project.Title,
			project.CreatorURL,
			project.CreatorLogin,
			project.Number)
	}

	return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprintln(message.Chat.ID)),
		projectList)}
}

func (s *Root) GroupTextMessage(message update.GroupTextMessage) (Handler, []response.BotAction) {
	switch strings.ToLower(strings.TrimSpace(message.Text)) {
	case "/start":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Start)}
	case "/help":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Help)}
	case "/addapikey", "/listprojects":
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
