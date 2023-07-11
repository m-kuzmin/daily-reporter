package state

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/m-kuzmin/daily-reporter/internal/clients/github"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
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
	return NewTransition(s.DailyStatusState, s.userData, []response.BotAction{
		response.CallbackQueryAnswerNotification(callback.ID, "This button doesnt work. Use /cancel to quit /dailyStatus."),
	})
}

func (s *DailyStatusHandler) Ignore() Transition {
	return NewTransition(s.DailyStatusState, s.userData, response.Nothing())
}

//nolint:cyclop
func (s *DailyStatusHandler) handleDailyStatus(chatID update.ChatID, text string) Transition {
	cmd, isCmd := slashcmd.Parse(text)

	if isCmd && strings.ToLower(cmd.Method) == cancelCommand {
		return NewTransition(s.RootState, s.userData, []response.BotAction{
			response.NewSendMessage(chatID, "Canceled."),
		})
	}

	if s.userData.GithubAPIKey.IsNone() {
		return NewTransition(s.RootState, s.userData, []response.BotAction{
			response.NewSendMessage(chatID, s.responses.NoAPIKeyAdded),
		})
	}

	switch s.Stage {
	case discoveryOfTheDayDailyStatusStage:
		s.DailyStatusState.Stage = questionsAndBlockersDailyStatusStage

		if isCmd && strings.ToLower(cmd.Method) == noneCommand {
			s.DiscoveryOfTheDay = option.None[string]()
		} else {
			s.DiscoveryOfTheDay = option.Some(text)
		}

		return NewTransition(s.DailyStatusState, s.userData, []response.BotAction{
			response.NewSendMessage(chatID,
				s.responses.QuestionsAndBlockers,
			),
		})

	case questionsAndBlockersDailyStatusStage:
		if isCmd && strings.ToLower(cmd.Method) == noneCommand {
			s.QuestionsAndBlockers = option.None[string]()
		} else {
			s.QuestionsAndBlockers = option.Some(text)
		}

		defaultProject, isSome := s.DefaultProject.Unwrap()
		if !isSome {
			return NewTransition(s.RootState, s.userData, []response.BotAction{
				response.NewSendMessage(chatID, s.responses.UseSetDefaultProject),
			})
		}

		report, err := s.generateReport(defaultProject)
		if err != nil {
			report = github.GqlErrorStringOr("GitHub API error: %s", err, s.responses.GithubErrorGeneric)
		}

		return NewTransition(s.RootState, s.userData, []response.BotAction{
			response.NewSendMessage(chatID, report),
		})
	}

	return s.Ignore()
}

func (s *DailyStatusHandler) generateReport(projectID github.ProjectID) (string, error) {
	items, err := github.NewClient(s.userData.GithubAPIKey.UnwrapOr("")).ListViewerProjectV2Items(context.Background(),
		projectID, dailyStatusItemLimit, option.None[github.ProjectCursor]())
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
	RootState
}

func NewDailyStatusState(root RootState) DailyStatusState {
	return DailyStatusState{
		Stage:                discoveryOfTheDayDailyStatusStage,
		DiscoveryOfTheDay:    option.None[string](),
		QuestionsAndBlockers: option.None[string](),
		RootState:            root,
	}
}

type dailyStatusStage int

const (
	discoveryOfTheDayDailyStatusStage dailyStatusStage = iota
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
	NoAPIKeyAdded        string `template:"noApiKeyAdded"`
	UseSetDefaultProject string `template:"useSetDefaultProject"`
}
