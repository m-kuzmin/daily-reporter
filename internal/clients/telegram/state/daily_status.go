package state

import (
	"fmt"
	"strings"

	"github.com/m-kuzmin/daily-reporter/internal/clients/github"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/template"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
	"github.com/m-kuzmin/daily-reporter/internal/util/slashcmd"
)

type DailyStatusHandler struct {
	responses *DailyStatusResponses
	userData  UserSharedData
	DailyStatusState
}

func (s *DailyStatusHandler) GroupTextMessage(message update.GroupTextMessage) Transition {
	cmd, isCmd := slashcmd.Parse(message.Text)

	if isCmd && strings.ToLower(cmd.Method) == "cancel" {
		return NewTransition(s.RootState, s.userData, []response.BotAction{
			response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)), "Canceled."),
		})
	}

	switch s.Stage {
	case ignoreMessagesDailyStatusStage:
		break

	case discoveryOfTheDayDailyStatusStage:
		s.DailyStatusState.Stage = questionsAndBlockersDailyStatusStage

		if isCmd && strings.ToLower(cmd.Method) == noneCommand {
			s.DiscoveryOfTheDay = option.None[string]()
		} else {
			s.DiscoveryOfTheDay = option.Some(message.Text)
		}

		return NewTransition(s.DailyStatusState, s.userData, []response.BotAction{
			response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)),
				s.responses.QuestionsAndBlockers,
			),
		})

	case questionsAndBlockersDailyStatusStage:
		if isCmd && strings.ToLower(cmd.Method) == noneCommand {
			s.QuestionsAndBlockers = option.None[string]()
		} else {
			s.QuestionsAndBlockers = option.Some(message.Text)
		}

		return NewTransition(s.RootState, s.userData, []response.BotAction{
			response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)), s.generateReport()),
		})
	}

	return s.Ignore()
}

func (s *DailyStatusHandler) PrivateTextMessage(update.PrivateTextMessage) Transition {
	return s.Ignore()
}

func (s *DailyStatusHandler) CallbackQuery(callback update.CallbackQuery) Transition {
	ignoreWithAlert := NewTransition(s.RootState, s.userData, []response.BotAction{
		response.CallbackQueryAnswerNotification(callback, "This button doesnt work."),
	})

	data, isSome := callback.Data.Unwrap()
	if !isSome {
		return ignoreWithAlert
	}

	message, isSome := callback.Message.Unwrap()
	if !isSome {
		return ignoreWithAlert
	}

	if transition, handled := s.handleCQSaveDefaultProject(data, message); handled {
		return transition
	}

	return ignoreWithAlert
}

func (s *DailyStatusHandler) Ignore() Transition {
	return NewTransition(s.DailyStatusState, s.userData, response.Nothing())
}

/*
Checks if the query is to save the default project. The retuned bool indicates if the query was about saving the default
project. If returns false try other queries if present or return an ignore response.
*/
func (s *DailyStatusHandler) handleCQSaveDefaultProject(data string, message update.Message) (Transition, bool) {
	askDiscoveryOfTheDay := []response.BotAction{
		response.RemoveReplyMarkup(message),
		response.NewSendMessage(response.ChatID(fmt.Sprint(message.Chat.ID)), s.responses.DiscoveryOfTheDay),
	}

	if data == cqDailyStatusAskDefaultProjectEveryTime {
		s.UseOnlyProjectNoSaveDefault = true

		return NewTransition(s.DailyStatusState, s.userData, askDiscoveryOfTheDay), true
	}

	if data == cqDailyStatusSetOnlyProjectAsDefault {
		s.DefaultProject = option.Some[github.ProjectID](s.UseProject)

		return NewTransition(s.DailyStatusState, s.userData, askDiscoveryOfTheDay), true
	}

	return Transition{}, false
}

func (s *DailyStatusHandler) generateReport() string {
	return "Imagine this is your report"
}

type DailyStatusState struct {
	Stage                dailyStatusStage
	DiscoveryOfTheDay    option.Option[string]
	QuestionsAndBlockers option.Option[string]
	UseProject           github.ProjectID // The project we are generating report for
	RootState
}

/*
Ignores user's messages until they click a button to optionaly save the only project they have as the default for this
chat.
*/
func NewDailyStatusStateAskSaveDefault(state RootState, project github.ProjectID) DailyStatusState {
	return DailyStatusState{
		Stage:                ignoreMessagesDailyStatusStage,
		DiscoveryOfTheDay:    option.None[string](),
		QuestionsAndBlockers: option.None[string](),
		UseProject:           project,
		RootState:            state,
	}
}

// The next message in the chat will be the answer for the first question in the sequence.
func NewDailyStatusStateForProject(state RootState, project github.ProjectID) DailyStatusState {
	return DailyStatusState{
		Stage:                discoveryOfTheDayDailyStatusStage,
		DiscoveryOfTheDay:    option.None[string](),
		QuestionsAndBlockers: option.None[string](),
		UseProject:           project,
		RootState:            state,
	}
}

type dailyStatusStage int

const (
	ignoreMessagesDailyStatusStage dailyStatusStage = iota
	discoveryOfTheDayDailyStatusStage
	questionsAndBlockersDailyStatusStage
)

func (s DailyStatusState) Handler(userData UserSharedData, responses *Responses) Handler {
	return &DailyStatusHandler{
		responses:        &responses.DailyStatus,
		userData:         userData,
		DailyStatusState: s,
	}
}

type DailyStatusResponses struct {
	DiscoveryOfTheDay    string `template:"discoveryOfTheDay"`
	QuestionsAndBlockers string `template:"questionsAndBlockers"`
}

func newDailyStatusResponse(template template.Template) (DailyStatusResponses, error) {
	group, err := template.Get("dailyStatus")
	if err != nil {
		return DailyStatusResponses{}, fmt.Errorf(`while getting "dailyStatus" group from template: %w`, err)
	}

	resp := DailyStatusResponses{}

	err = group.Populate(&resp)
	if err != nil {
		return DailyStatusResponses{}, fmt.Errorf(`while populating DailyStatusResponses from template: %w`, err)
	}

	return resp, nil
}
