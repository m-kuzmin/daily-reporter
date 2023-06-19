package state

import (
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/template"
)

type Handler interface {
	PrivateTextMessage(update.PrivateTextMessage) (Handler, []response.BotAction)
	GroupTextMessage(update.GroupTextMessage) (Handler, []response.BotAction)
	SetTemplate(template.Template) error
}
