package state

/*
Responses holds parsed and ready to use responses for all states. You can be sure no state uses a response not in this
struct because of `ConversationState` interface.
*/
type Responses struct {
	Root              rootResponses              `template:"root"`
	AddAPIKey         addAPIKeyResponses         `template:"addApiKey"`
	DailyStatus       DailyStatusResponses       `template:"dailyStatus"`
	SetDefaultProject SetDefaultProjectResponses `template:"setDefaultProject"`
}
