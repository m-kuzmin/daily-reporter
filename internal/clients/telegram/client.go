package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/state"
	"github.com/m-kuzmin/daily-reporter/internal/clients/telegram/update"
	"github.com/m-kuzmin/daily-reporter/internal/template"
)

const (
	getUpdatesLimit              = "20" // How many updates should telegram API send to us
	getUpdatesLongPollingTimeout = "5"  // The server will wait this many sec before telling us there's nothing to process
	doAPIRequestRetries          = 3    // After this many failures stop trying again
	getUpdatesRetries            = 10   // After this many failures stop trying again
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
	host     string // URL of the API server, without `https://`. E.g. `api.telegram.org`
	basePath string // `basePath + endpointPath` for making requests. Constructed from `"bot"+token`
	client   http.Client
	wg       sync.WaitGroup // Used to make sure all processor threads are done
	// When the bot crashes instead of paniking and crashing the whole app it sends the error here
	errCh              chan<- error
	stopProcessing     context.CancelFunc // Triggers the shutdown
	conversationStates map[string]state.Handler
	template           template.Template

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
		host:     host,
		basePath: "bot" + token,
		client:   http.Client{},
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
	errCh := make(chan error)
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
	}

	c.bot = botUser
	log.Printf("Bot's user: %#v", botUser)

	var (
		updateCh = make(chan update.Update, threads)
		stateCh  = make(chan updateWithState)
	)

	c.conversationStates = make(map[string]state.Handler)

	go c.getUpdates(ctx, updateCh)
	go c.stateQueue(updateCh, stateCh)

	for i := uint(0); i < threads; i++ {
		go c.processUpdates(stateCh)
	}

	return errCh
}

// GetMe returns a user that represents this bot
func (c *Client) GetMe(ctx context.Context) (update.User, error) {
	req, err := c.NewGetRequest(ctx, "getMe", url.Values{}, nil)
	if err != nil {
		return update.User{}, err
	}

	botUser, err := doAPIRequest[update.User](c, req)
	if err != nil {
		return update.User{}, err
	}

	return botUser, nil
}

// updateWithState is used to join an update with conversation state for that update.
type updateWithState struct {
	update update.Update // The update itself
	state  state.Handler // Conversation state
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
//nolint:funlen,cyclop // The control flow is just a sequence of error handlers
func (c *Client) getUpdates(ctx context.Context, updateCh chan<- update.Update) {
	c.wg.Add(1)

	shutdown := func() {
		close(updateCh)
		c.wg.Done()
	}

	query := url.Values{} // Stores the update id offset, so should not reset between iterations

	query.Add("limit", getUpdatesLimit)
	query.Add("timeout", getUpdatesLongPollingTimeout)

	log.Println("Telegram bot processor started")

	failures := 0
	for failures < getUpdatesRetries {
		select {
		case <-ctx.Done():
			close(updateCh)
			c.wg.Done()

			return
		default:
			req, err := c.NewGetRequest(ctx, "getUpdates", query, nil)
			if err != nil {
				shutdown()
				c.fail(fmt.Errorf("failed to prepare /getUpdates request: %w", err))

				return
			}

			result, err := doAPIRequest[[]update.Update](c, req)
			if err != nil {
				var apiErr apiError
				if ok := errors.As(err, &apiErr); ok && apiErr.ErrorCode == http.StatusUnauthorized {
					shutdown()
					c.fail(fmt.Errorf("bot token is likely invalid: %w", apiErr))

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

			for i, upd := range result {
				log.Printf("Sending update #%d to the queue", upd.ID)
				updateCh <- (result)[i]
			}

			// Sets the offset to the last update's ID
			if len(result) >= 1 {
				lastUpdate := result[len(result)-1]
				query.Set("offset", strconv.FormatInt(int64(lastUpdate.ID)+1, 10))
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
func (c *Client) stateQueue(updateCh chan update.Update, stateCh chan<- updateWithState) {
	c.wg.Add(1)

	// Cannot use normal defer here because of call to c.fail().
	defer func() {
		if err := recover(); err != nil {
			c.wg.Done()
			close(stateCh)
			log.Panicf("Caught panic in stateQueue: %s", err)
		}
	}()

	// TODO: Hold back updates that could cause race conditions in parallel processing
	for upd := range updateCh {
		upd := upd // creates a copy

		state, err := c.takeState(upd)
		if err != nil {
			c.wg.Done()
			close(stateCh)
			c.fail(err)

			return
		}

		stateCh <- updateWithState{
			update: upd,
			state:  state,
		}
	}

	c.wg.Done()
	close(stateCh)
}

/*
TODO(takeState, releaseState): Once takeState gives out an instance of state for a conversation it should not be able to
give out another instance. If there are 2 things that hold on to state to the same conversation they could try to update
it (save it to state storage) which would result in one state being lost and the final state is whatever was saved last.
They could also be operating on invalid state.

Consider 2 updates: /quiz and then /start. And lets say /quiz makes the bot send a question and the next message (in
this case /start) would be the answer. If we process /quiz and /start at the same time /start update would have no idea
that it should be handled as an answer to /quiz, and the user will get 2 messages: the first one is the quiz question
and the second one is the greeting message from /start. Then the user tries to answer the quiz and nothing happens
because /start saved the state last and overwrote what /quiz did.

TODO: If the state for a conversation was given out and not released via releaseState() this should fail or block.
*/
func (c *Client) takeState(u update.Update) (state.Handler, error) {
	handle, isSome := u.StateID().Unwrap()
	if isSome {
		if state, found := c.conversationStates[handle]; found {
			err := state.SetTemplate(c.template)
			if err != nil {
				log.Printf("While fetching state from storage for handle %q: %s", handle, err)
			} else {
				log.Printf("Lent state %T", state)

				return state, nil
			}
		}
	}

	state := &state.Root{}

	if err := state.SetTemplate(c.template); err != nil {
		return state, fmt.Errorf("while fetching state from storage for the default state: %w", err)
	}

	log.Printf("Lent state %T", state)

	return state, nil
}

// TODO: Should unlock the lock that  prevents takeState() from giving out state for the conversation.
func (c *Client) releaseState(u update.Update, s state.Handler) {
	if handle, isSome := u.StateID().Unwrap(); !isSome {
		log.Printf("No state handle for an update %#v", u)
	} else {
		log.Printf("Released state: %T", s)
		c.conversationStates[handle] = s
	}
}

/*
processUpdates method should be run in a goroutine and will process updates that come through the channel.

Stop this goroutine by closing the channel.
*/
func (c *Client) processUpdates(updateWithStateCh <-chan updateWithState) {
	c.wg.Add(1)

	for job := range updateWithStateCh {
		state, actions := state.Handle(c.bot, job.update, job.state)
		for _, action := range actions {
			endpoint, query := action.URLEncode()

			req, err := c.NewGetRequest(context.Background(), endpoint, query, nil) // Use context from Client
			if err != nil {
				c.wg.Done()
				c.fail(err)

				return
			}

			_, err = doAPIRequest[struct{}](c, req)
			if err != nil {
				// TODO: Stop if too many errors
				log.Printf("Error while performing /%s: %s\n", endpoint, err)
			}
		}

		c.releaseState(job.update, state)
	}

	c.wg.Done()
}

// Makes a GET method request from components.
func (c *Client) NewGetRequest(
	ctx context.Context, endpoint string, query url.Values, body io.Reader,
) (*http.Request, error) {
	url := url.URL{
		Scheme:   "https",
		Host:     c.host,
		Path:     path.Join(c.basePath, endpoint),
		RawQuery: query.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), body)
	if err != nil {
		// Delegates the correctness of the request to the one who is making it. If they can't ensure the request will
		// be created, they should do it themselves.
		return nil, fmt.Errorf("while constructing get request to /%s: %w", endpoint, err)
	}

	return req, nil
}

/*
Performs the Telegram API request, does error handling, wraps errors and then returns the payload of the request.
*/
func doAPIRequest[T any](client *Client, req *http.Request) (T, error) {
	var (
		lastErr    error
		zeroValOfT T
	)

	for i := 0; i < doAPIRequestRetries; i++ {
		resp, err := client.client.Do(req)
		if err != nil {
			return zeroValOfT, fmt.Errorf("network error: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()

			return zeroValOfT, fmt.Errorf("could not read response body %w", err)
		}

		resp.Body.Close()

		var data struct {
			Ok          bool   `json:"ok"`
			ErrorCode   int    `json:"error_code,omitempty"`
			Description string `json:"description,omitempty"`
			Parameters  struct {
				MigrateToChatID int64 `json:"migrate_to_chat_id,omitempty"`
				RertyAfter      int   `json:"retry_after,omitempty"`
			} `json:"parameters,omitempty"`
			Result T `json:"result,omitempty"`
		}

		if err = json.Unmarshal(body, &data); err != nil {
			return data.Result, fmt.Errorf("parsing json response error: %w", err)
		}

		if !data.Ok {
			err = apiError{ErrorCode: data.ErrorCode, Description: data.Description}
			lastErr = err

			if data.ErrorCode == http.StatusUnauthorized {
				return zeroValOfT, err
			}

			if data.Parameters.MigrateToChatID != 0 {
				return zeroValOfT, err
			}

			if data.Parameters.RertyAfter != 0 {
				time.Sleep(time.Duration(data.Parameters.RertyAfter) * time.Second)

				continue
			}

			continue
		}

		return data.Result, nil
	}

	return zeroValOfT, retryError{Retries: doAPIRequestRetries, LastError: lastErr}
}

// apiError from the telegram API.
type apiError struct {
	ErrorCode   int    // Error code (usually can be interpreted as HTTP response codes)
	Description string // Human readable explanation of the status code
}

func (e apiError) Error() string {
	return fmt.Sprintf("telegram API error: %d: %q", e.ErrorCode, e.Description)
}

/*
retryError means that there were attempts to try again, but the limit was exceeded and the
operation is cosidered as failed.
*/
type retryError struct {
	Retries   int   // How many times the action was performed before giving up
	LastError error // Latest error recorded from the API
}

func (e retryError) Error() string {
	return fmt.Sprintf("too many telegram API errors, retry limit (%d) exceeded; last error: %s", e.Retries, e.LastError)
}
