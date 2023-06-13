package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"sync"
	"time"
)

const (
	getUpdatesLimit              = "20" // How many updates should telegram API send to us
	getUpdatesLongPollingTimeout = "5"  // The server will wait this many sec before telling us there's nothing to process
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
	host           string // URL of the API server, without `https://`. E.g. `api.telegram.org`
	basePath       string // `basePath + endpointPath` for making requests. Constructed from `"bot"+token`
	client         http.Client
	wg             sync.WaitGroup     // Used to make sure all processor threads are done
	stopProcessing context.CancelFunc // Triggers the shutdown
}

/*
NewClient creates a new client.

`host` is the address to the server, without `https://`

`token` is the bot token for the API (Get it from @BotFather)

Creating the client is not enough, you have to `Start()` it.
*/
func NewClient(host, token string) Client {
	return Client{
		host:     host,
		basePath: "bot" + token,
		client:   *http.DefaultClient,
	}
}

/*
Start starts the client in the background. This function is non-blocking, meaning you dont have to
execute it in a goroutine (also look into `Stop()`).

`goroutines` specifies how many threads to use for processing telegram updates. Refer to
/docs/telegram-client/README.md for explanation of what these goroutines are and what they do.

This function creates a few goroutines inside. The first one is for fetching updates from Telegram.
There are also `goroutines` amount (argument to this func) of processor goroutines. These are used
to process multiple updates at the same time.

If we didnt process updates in parallel, then things that should be fast (like /start command)
would have to wait in a queue. If in the same queue there is a "heavy" update that does a lot of
things (e.g. read from database, query the API, etc) then all the quick-to-handle updates will just
sit there.

These heavy updates are heavy because they take a lot of time. But most likely that isn't because
they do some crazy computation. More likely because they are blocked by some IO. So all they are
really doing is idling.

Instead of wasting that idle time, a different goroutine could be executed.

# Panics

This function requires the number of goroutines to be at least one.
*/
func (c *Client) Start(goroutines uint) {
	if goroutines == 0 {
		panic(fmt.Sprintf(
			"Should never start a telegram bot with less than 1 processor threads. Was asked to use %d threads.",
			goroutines))
	}

	var (
		updateCh    = make(chan UpdateProcessor, goroutines)
		stateCh     = make(chan updateHandler)
		ctx, cancel = context.WithCancel(context.Background())
	)

	c.stopProcessing = cancel

	go c.getUpdates(ctx, updateCh)
	go c.stateQueue(updateCh, stateCh)

	for i := uint(0); i < goroutines; i++ {
		go c.processUpdates(stateCh)
	}
}

// updateHandler is used to join an update with conversation state for that update.
type updateHandler struct {
	processor UpdateProcessor          // The update itself
	state     ConversationStateHandler // Conveersation state
}

/*
Stop stops the telegram client gracefully. This function returns after the server
is no longer doing any work.

Call this function when your application shutsdown (aka `defer` it in main)

Do keep in mind that your main cant look like this (or equvalent):

		client.Start(1)
		client.Stop() // instantly call Stop

		// this is bad too
	    defer client.Stop()
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
getUpdates method should be run in a goroutine and will call `/getUpdates`
telegram API endpoint, parse the incoming updates and send them to the
queue for processing.

After an update has been fetched and sent to the queue its considered
as processed by the telegram API.

# Panics

This function will panic if it cannot construct a valid API URL. To avoid this make sure
the parameters to Client's constructor are valid.

It also panics if there were too many errors while talking to the telegram API
*/
func (c *Client) getUpdates(ctx context.Context, updateCh chan<- UpdateProcessor) {
	c.wg.Add(1)
	defer c.wg.Done()
	defer close(updateCh)

	query := url.Values{} // Stores the update id offset, so should not reset between iterations

	query.Add("limit", getUpdatesLimit)
	query.Add("timeout", getUpdatesLongPollingTimeout)

	log.Println("Telegram bot processor started")

	const retry = 10

	failures := 0
	for failures < retry {
		select {
		case <-ctx.Done():
			return
		default:
			req := c.mustNewGetRequest(ctx, "getUpdates", query, nil)

			result, err := doAPIRequest[[]update](c, req)
			if err != nil {
				failures++
				log.Printf("/getUpdates failure (try %d): %s\n", failures, err)

				continue
			}

			failures = 0

			log.Println("/getUpdates failure count reset to 0")

			for i, upd := range *result {
				log.Printf("Sending update #%d to the queue", upd.ID)

				// upd is not ok here because of https://stackoverflow.com/questions/62446118/implicit-memory-aliasing-in-for-loop
				updateCh <- &(*result)[i]
			}

			// Prevents new updates from being the same thing
			if len(*result) >= 1 {
				lastUpdate := (*result)[len(*result)-1]
				query.Set("offset", strconv.FormatInt(lastUpdate.ID+1, 10))
			}
		}
	}
	panic(fmt.Sprintf("Bot encountered too many errors (%d) while interacting with Telegram API", retry))
}

/*
stateQueue manages conversation state. It should be run in a goroutine. The job of this
function is to take updates from `updateCh`, combine them with conversation state and send
that to `stateCh`.

However it is not as simple as that. After an update is processed it could change the state
of the conversation. If two updates try to use and change the same state they could create
weird bugs. Refer to /docs/telegram-client/README.md for details.
*/
func (c *Client) stateQueue(updateCh <-chan UpdateProcessor, stateCh chan<- updateHandler) {
	c.wg.Add(1)
	defer c.wg.Done()

	defer close(stateCh)

	// TODO: Hold back updates that could cause race conditions in parallel processing
	for upd := range updateCh {
		stateCh <- updateHandler{
			processor: upd,
			state:     c.takeState(),
		}
	}
}

/*
TODO: should return state for the update that will be processed.
Should also block if the state is taken already
(Should take some args to look up the state for the conversation)
*/
func (c *Client) takeState() ConversationStateHandler { //nolint:ireturn
	return &rootConversationState{}
}

// TODO: Stores the state and allows other threads to take it via takeState
func (c *Client) releaseState(ConversationStateHandler) {}

/*
processUpdates should be run in a goroutine and will process updates
that come through the channel.

Stop this goroutine by closing the channel
*/
func (c *Client) processUpdates(updateCh <-chan updateHandler) {
	c.wg.Add(1)
	defer c.wg.Done()

	for update := range updateCh {
		state, actions := update.processor.processTelegramUpdate(update.state)
		for _, action := range actions {
			endpoint, query := action.telegramBotAction()

			req := c.mustNewGetRequest(context.Background(), endpoint, query, nil)

			_, err := doAPIRequest[struct{}](c, req)
			if err != nil {
				log.Printf("Error while performing /%s: %s\n", endpoint, err)
			}
		}

		c.releaseState(state)
	}
}

// Makes a GET method request from components. Panics if the request cannot be created
func (c *Client) mustNewGetRequest(
	ctx context.Context,
	endpoint string,
	query url.Values,
	body io.Reader,
) *http.Request {
	url := url.URL{
		Scheme:   "https",
		Host:     c.host,
		Path:     path.Join(c.basePath, endpoint),
		RawQuery: query.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url.String(), body)
	if err != nil {
		// panic because a request should always be valid
		log.Fatal("Error: Request should always be valid, %w", err)
	}

	return req
}

/*
doApiRequest performs the Telegram API request, does error handling, wraps errors
and then returns the payload of the request

# Panics

doApiRequest Panics if the status code is 401 Unauthorized which is usually caused by an invalid bot token.

Or if the telegram API sends a "Migrate to chat ID" error, which is unsupported as of now and could cause weird bugs.

Will also panic if the telegram API returns an error many timess in a row (probably 10 times,
but look at the source to see the exact value.
*/
func doAPIRequest[T any](c *Client, req *http.Request) (*T, error) {
	const retry = 3
	for i := 0; i < retry; i++ {
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("network error: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return nil, fmt.Errorf("could not read response body: %w", err)
		}

		// Parse

		var data struct {
			Ok          bool   `json:"ok"`
			ErrorCode   int    `json:"error_code,omitempty"`
			Description string `json:"description,omitempty"`
			Parameters  struct {
				MigrateToChatID int64 `json:"migrate_to_chat_id,omitempty"`
				RertyAfter      int   `json:"retry_after,omitempty"`
			} `json:"parameters,omitempty"`
			Result T `json:"result"`
		}

		if err = json.Unmarshal(body, &data); err != nil {
			return nil, fmt.Errorf("parsing json response error: %w", err)
		}

		if !data.Ok {
			err = apiError{ErrorCode: data.ErrorCode, Description: data.Description}
			log.Println(err)

			if data.ErrorCode == http.StatusUnauthorized {
				log.Fatal("Token is likely invalid")
			}

			if data.Parameters.RertyAfter != 0 {
				time.Sleep(time.Duration(data.Parameters.RertyAfter) * time.Second)

				continue
			}

			if data.Parameters.MigrateToChatID != 0 {
				log.Fatalf("API asked to migrate to chat ID %d, which is an unsupported operation",
					data.Parameters.MigrateToChatID)
			}

			continue
		}

		return &data.Result, nil
	}

	return nil, retryError{Retries: retry}
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
	Retries int // How many times the action was performed before giving up
}

func (e retryError) Error() string {
	return fmt.Sprintf("too many telegram API errors, retry limit (%d) exceeded.", e.Retries)
}
