package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/m-kuzmin/daily-reporter/internal/clients/github"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/template"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
	"github.com/m-kuzmin/daily-reporter/internal/util/slashcmd"
	"github.com/pkg/errors"
)

const dailyStatusItemLimit = 100

type DailyStatusHandler struct {
	responses *DailyStatusResponses
	userData  UserSharedData
	DailyStatusState
}

func (s *DailyStatusHandler) GroupTextMessage(message update.GroupTextMessage) Transition {
	return s.handleDailyStatus(message.Chat.ID, message.Text)
}

func (s *DailyStatusHandler) PrivateTextMessage(message update.PrivateTextMessage) Transition {
	return s.handleDailyStatus(message.Chat.ID, message.Text)
}

func (s *DailyStatusHandler) CallbackQuery(callback update.CallbackQuery) Transition {
	ignoreWithAlert := NewTransition(s.RootState, s.userData, []response.BotAction{
		response.CallbackQueryAnswerNotification(callback.ID, "This button doesnt work."),
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

//nolint:cyclop
func (s *DailyStatusHandler) handleDailyStatus(chatID update.ChatID, text string) Transition {
	cmd, isCmd := slashcmd.Parse(text)

	if isCmd && strings.ToLower(cmd.Method) == cancelCommand {
		return NewTransition(s.RootState, s.userData, []response.BotAction{
			response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)), "Canceled."),
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
			s.DiscoveryOfTheDay = option.Some(text)
		}

		return NewTransition(s.DailyStatusState, s.userData, []response.BotAction{
			response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)),
				s.responses.QuestionsAndBlockers,
			),
		})

	case questionsAndBlockersDailyStatusStage:
		if isCmd && strings.ToLower(cmd.Method) == noneCommand {
			s.QuestionsAndBlockers = option.None[string]()
		} else {
			s.QuestionsAndBlockers = option.Some(text)
		}

		report, err := s.generateReport()
		if err != nil {
			report = github.GqlErrorStringOr("Github API error: %s", err, "Something went wrong while contacting GitHub.")
		}

		return NewTransition(s.RootState, s.userData, []response.BotAction{
			response.NewSendMessage(response.ChatID(fmt.Sprint(chatID)), report),
		})
	}

	return s.Ignore()
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

func (s *DailyStatusHandler) generateReport() (string, error) {
	items, err := github.NewClient(s.userData.GithubAPIKey.UnwrapOr("")).ListViewerProjectV2Items(context.Background(),
		s.UseProject, option.Some(dailyStatusItemLimit), option.None[github.ProjectCursor]())
	if err != nil {
		return "", errors.WithMessage(err, "while getting user's project v2 items")
	}

	const listSep = "\n• "

	report := fmt.Sprintf(`#daily report %s:
<b>Today I worked on</b>%s

<b>Tomorrow I will work on</b>%s

`, time.Now().Format("01.02"), listSep+strings.Join(items["Done"], listSep),
		listSep+strings.Join(items["In Progress"], listSep))

	if dod, isSome := s.DiscoveryOfTheDay.Unwrap(); isSome {
		report += "<b>Discovery of the day</b>\n\n" + dod + "\n\n"
	}

	if blockers, isSome := s.QuestionsAndBlockers.Unwrap(); isSome {
		report += "<b>Questions/Blockers</b>\n\n" + blockers + "\n\n"
	}

	if len(items["In Review"]) != 0 {
		report += "<b>In review</b>" + listSep + strings.Join(items["In Review"], listSep)
	}

	return report, nil
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
	GithubErrorGeneric   string `template:"githubErrorGeneric"`
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
