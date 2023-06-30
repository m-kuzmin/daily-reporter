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

const listProjectsCommand = "listprojects"

func (s *Root) PrivateTextMessage(message update.PrivateTextMessage) (Handler, []response.BotAction) {
	cmd, isCmd := slashcmd.Parse(message.Text)
	if !isCmd {
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.UnknownMessage)}
	}

	switch strings.ToLower(cmd.Method) {
	case "start":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Start)}
	case "help":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Help)}
	case "addapikey":
		return &AddAPIKey{prevState: *s}, []response.BotAction{response.NewSendMessage(
			response.ChatID(fmt.Sprint(message.Chat.ID)), s.responses.AddAPIKey)}
	case listProjectsCommand:
		after := option.None[github.ProjectCursor]()
		if afterV, hasAfter := cmd.NextAfter("after"); hasAfter && afterV != "" {
			after = option.Some(github.ProjectCursor(afterV))
		}

		return s.handleListProjects(message.Chat.ID, after)
	}

	return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
		s.responses.UnknownMessage)}
}

func (s *Root) handleListProjects(
	chatID update.ChatID,
	afterCursor option.Option[github.ProjectCursor],
) (Handler, []response.BotAction) {
	const projectsOnPage = 10

	// Get the user's key
	key, isSome := s.userData.GithubAPIKey.Unwrap()
	if !isSome {
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)),
			s.responses.ListProjectsWithoutAPIKey)}
	}

	// Get the user's projects
	projects, err := github.NewClient(key).ListViewerProjects(option.Some(projectsOnPage), afterCursor)
	if err != nil {
		log.Printf("While requesting user's projects: %s", err)

		if gqlMessage, is := github.GqlErrorString(err); is {
			return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)),
				fmt.Sprintf("GitHub API error: %s", gqlMessage))}
		}

		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)),
			"Something went wrong while doing a GitHub API request")}
	}

	if len(projects) == 0 {
		if afterCursor.IsNone() {
			return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprintln(chatID)),
				s.responses.UserHasZeroProjects)}
		}

		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprintln(chatID)),
			s.responses.LastProjectsPage)}
	}

	// Print the projects
	projectList := fmt.Sprintf("Your projects (%d/page)", projectsOnPage)

	for _, project := range projects {
		projectList += fmt.Sprintf("\n\n<code>%s</code> <a href=%q><b>%s</b></a> (<a href=%q>%s</a>/%d)",
			project.Cursor, project.URL, project.Title, project.CreatorURL, project.CreatorLogin, project.Number)
	}

	resp := response.NewSendMessage(response.ChatID(fmt.Sprintln(chatID)), projectList)

	if len(projects) == projectsOnPage {
		resp = resp.SetReplyMarkup([][]response.InlineKeyboardButton{{
			response.InlineButtonSwitchQueryCurrentChat("Next page",
				fmt.Sprintf("/%s after %s", listProjectsCommand, projects[len(projects)-1].Cursor)),
		}})
	}

	return s, []response.BotAction{resp}
}

func (s *Root) GroupTextMessage(message update.GroupTextMessage) (Handler, []response.BotAction) {
	cmd, isCmd := slashcmd.Parse(message.Text)
	if !isCmd {
		return s, response.Nothing()
	}

	switch strings.ToLower(cmd.Method) {
	case "start":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Start)}
	case "help":
		return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Help)}
	case "addapikey", listProjectsCommand:
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
