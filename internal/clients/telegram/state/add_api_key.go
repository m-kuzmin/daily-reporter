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

type AddAPIKey struct {
	responses addAPIKeyResponses
}

type addAPIKeyResponses struct {
	KeySentInPublicChat string `template:"keySentInPublicChat"`
	BadAPIKey           string `template:"badApiKey"`
	Cancel              string `template:"cancel"`
	Success             string `template:"success"`
}

func (s *AddAPIKey) PrivateTextMessage(message update.PrivateTextMessage) (Handler, []response.BotAction) {
	switch strings.ToLower(strings.TrimSpace(message.Text)) {
	case "/cancel":
		return &Root{}, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
			s.responses.Cancel)}
	default:
		client := github.NewClient(message.Text)

		login, err := client.Login()
		if err != nil {
			return s, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
				s.responses.BadAPIKey)}
		}

		return &Root{userData: rootUserData{GithubAPIKey: option.Some(message.Text)}},
			[]response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
				fmt.Sprintf(s.responses.Success, login, login)).EnableWebPreview()}
	}
}

func (s *AddAPIKey) GroupTextMessage(message update.GroupTextMessage) (Handler, []response.BotAction) {
	return &Root{}, []response.BotAction{response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
		s.responses.KeySentInPublicChat)}
}

func (s *AddAPIKey) SetTemplate(template template.Template) error {
	group, err := template.Get("addApiKey")
	if err != nil {
		return fmt.Errorf(`while getting "addApiKey" group from template: %w`, err)
	}

	err = group.Populate(&s.responses)
	if err != nil {
		return fmt.Errorf(`while populating addApiKeyResponses from template: %w`, err)
	}

	return nil
}
