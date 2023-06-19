package state

import (
	"fmt"
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/template"
)

type Root struct {
	responses rootResponses
}

type rootResponses struct {
	Start              string `template:"start"`
	Help               string `template:"help"`
	PrivateCommandUsed string `template:"privateCommandUsed"`
	AddAPIKey          string `template:"addApiKey"`
	UnknownMessage     string `template:"unknownMessage"`
}

func (s *Root) PrivateTextMessage(message update.PrivateTextMessage) (Handler, []response.BotAction) {
	switch strings.ToLower(strings.TrimSpace(message.Text)) {
	case "/start":
		return s, []response.BotAction{response.SendMessageBuilder(message.Chat.ID, s.responses.Start)}
	case "/help":
		return s, []response.BotAction{response.SendMessageBuilder(message.Chat.ID, s.responses.Help)}
	case "/addapikey":
		return &AddAPIKey{}, []response.BotAction{response.SendMessageBuilder(message.Chat.ID,
			s.responses.AddAPIKey).ParseModeHTML()}
	default:
		return s, []response.BotAction{response.SendMessageBuilder(message.Chat.ID, s.responses.UnknownMessage)}
	}
}

func (s *Root) GroupTextMessage(message update.GroupTextMessage) (Handler, []response.BotAction) {
	switch strings.ToLower(strings.TrimSpace(message.Text)) {
	case "/start":
		return s, []response.BotAction{response.SendMessageBuilder(message.Chat.ID, s.responses.Start)}
	case "/help":
		return s, []response.BotAction{response.SendMessageBuilder(message.Chat.ID, s.responses.Help)}
	case "/addapikey":
		return s, []response.BotAction{response.SendMessageBuilder(message.Chat.ID, s.responses.PrivateCommandUsed)}
	default:
		return s, response.Noop()
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
