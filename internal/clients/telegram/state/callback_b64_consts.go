package state

/*
This file contains random b64 encoded strings. Their purpose is to be unique and serve as `sendMessage?callback_query`
parameters. This ensures that all inline keyboard buttons in telegram have unique callback data and that it should not
ever clash with any strings that have special encoding.

To generate a new callback string use an online generator (or python) and produce a b64 16 character long URL safe
string. Entropy quality doesnt matter here as the only property these strings should have is resistance to being decoded
as any future or current callback query encodings. Since it is impossible to predict the future all we can do is use
decent enough sources of randomness to get randomized outputs.
*/

const (
	// Next time /dailyStatus is used ask the user about their default project.
	cqDailyStatusAskDefaultProjectEveryTime = "klLTDp9jrdwcwZ4i541zDA"

	// Save the only project the user has as their default for this chat
	cqDailyStatusSetOnlyProjectAsDefault = "OkuVoB7l1nEu-cjEMvVADQ"
)
