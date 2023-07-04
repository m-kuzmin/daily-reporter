package state

import "github.com/m-kuzmin/daily-reporter/internal/template"

/*
Responses holds parsed and ready to use responses for all states. You can be sure no state uses a response not in this
struct because of `ConversationState` interface.
*/
type Responses struct {
	Root        rootResponses
	AddAPIKey   addAPIKeyResponses
	DailyStatus DailyStatusResponses
}

/*
NewResponses populates response objects from template. This ensures all strings and groups are in the template
in one place (this func).
*/
func NewResponses(templ template.Template) (Responses, error) {
	root, err := newRootResponses(templ)
	if err != nil {
		return Responses{}, err
	}

	addAPIKey, err := newAddAPIKeyResponse(templ)
	if err != nil {
		return Responses{}, err
	}

	dailyStatus, err := newDailyStatusResponse(templ)
	if err != nil {
		return Responses{}, err
	}

	return Responses{
		Root:        root,
		AddAPIKey:   addAPIKey,
		DailyStatus: dailyStatus,
	}, nil
}
