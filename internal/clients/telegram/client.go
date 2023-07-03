package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/response"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/state"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/template"
	"github.com/m-kuzmin/daily-reporter/internal/util"
	"github.com/m-kuzmin/daily-reporter/internal/util/borrowonce"
	"github.com/m-kuzmin/daily-reporter/internal/util/option"
)

const (
	getUpdatesLimit              = 20 // How many updates should telegram API send to us
	getUpdatesLongPollingTimeout = 5  // The server will wait this many sec before telling us there's nothing to process
	getUpdatesRetries            = 10 // After this many failures stop trying again
)

// Starter is a muiltithreaded client where the number of threads is passed into Start()
type Starter interface {
	Start(threads uint) // `threads` is the number of threads the client is allowed to use
	Stop()
}

/*
A Client to interact with the telegram API. Start the client and then call Stop to stop it gracefully.
The client may take some time to shutdown if it has work to do.

# Example

	// main.go

	client := telegram.NewClient("api.telegram.org", "TOKEN")

	client.Start(10) // 10 threads
	defer client.Stop()

	// some function that returns when the shutdown should happen
	blockUntilExitSignal()
*/
type Client struct {
	requester response.APIRequester

	wg sync.WaitGroup // Used to make sure all processor threads are done
	// When the bot crashes instead of paniking and crashing the whole app it sends the error here
	errCh          chan<- error
	stopProcessing context.CancelFunc // Triggers the shutdown

	conversationStateStore borrowonce.Storage[string, state.State]
	userSharedDataStore    borrowonce.Storage[update.UserID, state.UserSharedData]

	template template.Template

	bot update.User
}

/*
Creates a new client.

`host` is the address to the server, without `https://`

`token` is the bot token for the API

Creating the client is not enough, you have to `Start()` it.
*/
func NewClient(host, token string, template template.Template) Client {
	return Client{
		requester: response.APIRequester{
			Client:   http.Client{},
			Scheme:   "https",
			Host:     host,
			BasePath: "bot" + token,
		},
		template: template,
	}
}

/*
Start starts the client in the background. This function is non-blocking, meaning you dont have to
execute it in a goroutine (also look into `Stop()`).

`threads` specifies how many goroutines to use for processing telegram updates. Refer to
/docs/telegram-client/README.md for explanation of what these goroutines are and what they do.

Returned channel should be listened on. There can only be one value sent in this channel. This value is a fatal error
that signals that the bot crashed. This is a replacement to panic().

This function creates a few goroutines inside. The first one is for fetching updates from Telegram.
There are also `goroutines` amount (argument to this func) of processor goroutines. These are used
to process multiple updates at the same time
*/
func (c *Client) Start(threads uint) <-chan error {
	errCh := make(chan error, 1)
	c.errCh = errCh

	ctx, cancel := context.WithCancel(context.Background())
	c.stopProcessing = cancel

	if threads == 0 {
		//nolint:goerr113
		c.fail(fmt.Errorf(
			"should never start a telegram bot with less than 1 processor threads. Was asked to use %d threads",
			threads))

		return errCh
	}

	botUser, err := c.GetMe(ctx)
	if err != nil {
		c.fail(err)

		return errCh
	}

	c.bot = botUser

	var (
		updateCh = make(chan update.Update, 1)
		stateCh  = make(chan updateWithState, threads)
	)

	c.conversationStateStore = borrowonce.NewStorage[string, state.State]()
	c.userSharedDataStore = borrowonce.NewStorage[update.UserID, state.UserSharedData]()

	go c.getUpdates(ctx, updateCh)
	go c.stateQueue(updateCh, stateCh)

	for i := uint(0); i < threads; i++ {
		go c.processUpdates(ctx, stateCh)
	}

	return errCh
}

// GetMe returns a user that represents this bot
func (c *Client) GetMe(ctx context.Context) (update.User, error) {
	resp, err := c.requester.Do(ctx, "getMe", json.RawMessage{})
	if err != nil {
		return update.User{}, fmt.Errorf("while requesting /GetMe: %w", err)
	}

	var botUser update.User
	err = json.Unmarshal(resp, &botUser)

	return botUser, fmt.Errorf("while decoding /GetMe JSON response: %w", err)
}

// updateWithState is used to join an update with conversation state for that update.
type updateWithState struct {
	update   update.Update                            // The update itself
	state    *borrowonce.Future[state.State]          // Conversation state
	userData *borrowonce.Future[state.UserSharedData] // Data that should be shared across telegram chats
}

/*
Stop stops the telegram client gracefully. This function returns after the server
is no longer doing any work.

Call this function when your application shutsdown (aka `defer` it in main)

Do keep in mind that your main cant look like this (or equvalent):

	client.Start(1)

	client.Stop() // instantly call Stop

	// this is bad too ...
	defer client.Stop()
	// ... because function returns instantly
	return

Because then your telegram client is going to stop immediately after you started it.
Instead you can create a channel for SIGTERM (`Ctrl+C`) and `<-` on that.

	client.Start(1)
	defer client.Stop()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c // blocks until you do ^C in terminal
*/
func (c *Client) Stop() {
	c.stopProcessing()
	c.wg.Wait()
}

/*
fail stops the bot and allows the caller of Start() to know the bot crashed. This is a replacement to panics.

You can receive the errors from an error chan Start() gives back. You can assume the bot's functions are all finished by
the time you receive the error value.
*/
func (c *Client) fail(err error) {
	c.Stop()
	c.errCh <- err
}

/*
getUpdates method should be run in a goroutine and will call `/getUpdates` telegram API endpoint, parse the incoming
updates and send them to the queue for processing.

After an update has been fetched and sent to the queue its considered as processed by the telegram API.

When this function returns it also closes the channel effectively stopping all processor goroutines that are listening
on it.
*/
//nolint:funlen,cyclop // After refactoring it's still 70-ish lines :sad_emoji:.
func (c *Client) getUpdates(ctx context.Context, updateCh chan<- update.Update) {
	c.wg.Add(1)

	shutdown := func() {
		close(updateCh)
		c.wg.Done()
	}

	// Cannot use normal defer here because of call to c.fail().
	defer func() {
		if err := recover(); err != nil {
			shutdown()
			c.fail(fmt.Errorf("shutting down from getUpdates: %w", util.RecoveredPanicError{Panic: err}))
		}
	}()

	log.Println("Telegram bot processor started")

	getUpdates := getUpdatesRequest{
		Offset:  update.UpdateID(0),
		Limit:   getUpdatesLimit,
		Timeout: getUpdatesLongPollingTimeout,
	}

	failures := 0
	for failures < getUpdatesRetries {
		select {
		case <-ctx.Done():
			shutdown()

			return
		default:
			updates, err := getUpdates.Request(ctx, c.requester)
			if err != nil {
				var apiErr response.APIError
				if ok := errors.As(err, &apiErr); ok && apiErr.ErrorCode == http.StatusUnauthorized {
					shutdown()
					c.fail(fmt.Errorf("bot token is likely invalid: %w", apiErr))

					return
				} else if wait, isSome := apiErr.Parameters.RertyAfter.Unwrap(); isSome {
					time.Sleep(time.Duration(wait) * time.Second)

					continue
				} else if apiErr.Parameters.MigrateToChatID.IsSome() {
					shutdown()
					//nolint:goerr113
					c.fail(fmt.Errorf("telegram requsted to migrate to chage id, which is an unsupported operation"))

					return
				}

				failures++
				log.Printf("/getUpdates failure #%d: %s\n", failures, err)

				continue
			}

			if failures != 0 {
				log.Printf("/getUpdates failure count reset to 0")

				failures = 0
			}

			for i, upd := range updates {
				log.Printf("Sending update #%d to the queue", upd.ID)
				updateCh <- (updates)[i]

				if upd.ID > getUpdates.Offset {
					getUpdates.Offset = upd.ID
				}
			}
		}
	}

	shutdown()
	//nolint:goerr113
	c.fail(fmt.Errorf("bot encountered too many errors (%d) while interacting with Telegram API", getUpdatesRetries))
}

/*
stateQueue manages conversation state. It should be run in a goroutine. The job of this
function is to take updates from `updateCh`, combine them with conversation state and send
that to `stateCh`.

However it is not as simple as that. After an update is processed it could change the state
of the conversation. If two updates try to use and change the same state they could create
weird bugs. Refer to /docs/telegram-client/README.md for details.
*/
func (c *Client) stateQueue(updateCh <-chan update.Update, stateCh chan<- updateWithState) {
	c.wg.Add(1)

	shutdown := func() {
		c.wg.Done()
		close(stateCh)
	}

	// Cannot use normal defer here because of call to c.fail().
	defer func() {
		if err := recover(); err != nil {
			shutdown()
			c.fail(fmt.Errorf("shutting down from stateQueue: %w", util.RecoveredPanicError{Panic: err}))
		}
	}()

	// TODO: Hold back updates that could cause race conditions in parallel processing
	for upd := range updateCh {
		upd := upd // creates a copy

		futureState := borrowonce.NewImmediateFuture[state.State](&state.RootState{})
		futureUserData := borrowonce.NewImmediateFuture[state.UserSharedData](state.UserSharedData{
			GithubAPIKey: option.None[string](),
		})

		if handle, ok := upd.StateID(); ok {
			futureState = c.borrowState(handle)
		}

		if handle, ok := upd.UserID(); ok {
			futureUserData = c.borrowUserData(handle)
		}

		stateCh <- updateWithState{
			update:   upd,
			state:    futureState,
			userData: futureUserData,
		}
	}

	shutdown()
}

/*
borrowState returns a Future to access the latest value of the state. If no state is in the storage then assigns it to
Root.
*/
func (c *Client) borrowState(handle string) *borrowonce.Future[state.State] {
	if future, exists := c.conversationStateStore.Borrow(handle); exists {
		return future
	}

	c.conversationStateStore.Set(handle, &state.RootState{})

	if future, exists := c.conversationStateStore.Borrow(handle); exists {
		return future
	}

	// Implementation of borrownonce.Store guarantees that after a value is Set() it is available to borrow
	panic("conversation store did not lend a value after it was set explicitly")
}

/*
takeState returns a Future to access the latest value of the state. If no state is in the storage then assigns it to
Root.
*/
func (c *Client) borrowUserData(handle update.UserID) *borrowonce.Future[state.UserSharedData] {
	if future, exists := c.userSharedDataStore.Borrow(handle); exists {
		return future
	}

	c.userSharedDataStore.Set(handle, state.UserSharedData{
		GithubAPIKey: option.None[string](),
	})

	if future, exists := c.userSharedDataStore.Borrow(handle); exists {
		return future
	}

	// Implementation of borrownonce.Store guarantees that after a value is Set() it is available to borrow
	panic("conversation store did not lend a value after it was set explicitly")
}

/*
processUpdates method should be run in a goroutine and will process updates that come through the channel.

Stop this goroutine by closing the channel.
*/
func (c *Client) processUpdates(ctx context.Context, updateWithStateCh <-chan updateWithState) {
	c.wg.Add(1)

	shutdown := func() { c.wg.Done() }

	defer func() {
		if err := recover(); err != nil {
			shutdown()
			c.fail(fmt.Errorf("shutting down from processUpdates: %w", util.RecoveredPanicError{Panic: err}))
		}
	}()

	resp, err := state.NewResponses(c.template)
	if err != nil {
		shutdown()
		c.fail(fmt.Errorf("while constructing state.Responses in processUpdates: %w", err))

		return
	}

	for job := range updateWithStateCh {
		handler := job.state.Wait().Handler(job.userData.Wait(), &resp)

		transition := state.Handle(c.bot, job.update, handler)
		for _, action := range transition.Actions {
			endpoint, body, err := action.JSONEncode()
			if err != nil {
				log.Printf("Error while encoding an action to JSON: %s", err)

				continue
			}

			_, err = c.requester.Do(ctx, endpoint, body)
			if err != nil {
				log.Printf("Error while performing /%s: %s\n", endpoint, err)
			}
		}

		if id, ok := job.update.StateID(); ok {
			c.conversationStateStore.Return(id, transition.NewState)
		}

		if id, ok := job.update.UserID(); ok {
			c.userSharedDataStore.Return(id, transition.UserData)
		}
	}

	shutdown()
}

type getUpdatesRequest struct {
	Offset  update.UpdateID `json:"offset"`
	Limit   int64           `json:"limit"`
	Timeout int             `json:"timeout"`
}

func (r getUpdatesRequest) Request(ctx context.Context, requester response.APIRequester) ([]update.Update, error) {
	body, err := json.Marshal(r)
	if err != nil {
		return []update.Update{}, fmt.Errorf("while JSON serializing /getUpdates: %w", err)
	}

	body, err = requester.Do(ctx, "getUpdates", body)
	if err != nil {
		return []update.Update{}, fmt.Errorf("while requesting /getUpdates: %w", err)
	}

	var upd []update.Update

	err = json.Unmarshal(body, &upd)

	return upd, fmt.Errorf("while decoding /getUpdates JSON response: %w", err)
}
