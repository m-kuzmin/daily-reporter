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

// A client to interact with the telegram API. Start the client and then call Stop to stop it gracefully.
// The client may take some time to shutdown if it has work to do.
//
// TODO(doc): Add info from https://gist.github.com/m-kuzmin/f6675dad25fc74daacef3c7d0b5d2375
// # Example
//
//	client := telegram.NewClient("api.telegram.org", "TOKEN")
//	client.Start(10) // 10 threads
//
//	if shouldStop { // whatever the trigger for stop is
//	    client.Stop()
//	}
type Client struct {
	host           string // URL of the API server, without `https://`. E.g. `api.telegram.org`
	basePath       string // `basePath + endpointPath` for making requests. Constructed from `"bot"+token`
	client         http.Client
	wg             sync.WaitGroup     // Used to make sure all processor threads are done
	stopProcessing context.CancelFunc // Triggers the shutdown
}

// NewClient creates a new client.
//
// `host` is the address to the server, without `https://`
//
// `token` is the bot token for the API
func NewClient(host, token string) Client {
	return Client{
		host:     host,
		basePath: "bot" + token,
		client:   *http.DefaultClient,
	}
}

/*
Start starts the client in the background. It will fetch updates via `Client.getUpdates()`
and sends them to a queue. Queue items are received by one of `Client.processUpdates()`
which are going to process all updates in parallel

`goroutines` specifies how many threads to use in parallel.

# Panics

This function considers 0 or less goroutines as fatal.
*/
// TODO(doc): Explain what this func does and how to use it/how it works.
func (c *Client) Start(goroutines uint) {
	if goroutines < 1 {
		panic(fmt.Sprintf(
			"Should never start a telegram bot with less than 1 processor threads. Was asked to use %d threads.",
			goroutines))
	}

	var (
		updateQueue = make(chan UpdateProcessor, goroutines)
		stateQueue  = make(chan updateHandler)
		ctx, cancel = context.WithCancel(context.Background())
	)

	c.stopProcessing = cancel

	go c.getUpdates(ctx, updateQueue)
	go c.stateQueue(updateQueue, stateQueue)

	for i := uint(0); i < goroutines; i++ {
		go c.processUpdates(stateQueue)
	}
}

type updateHandler struct {
	processor UpdateProcessor
	state     ConversationStateHandler
}

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

Stop this goroutine by closing the stopCh. You can get an instance of stopCh
from `ctx.Done()` When this function returns it also closes the channel
effectively stopping all processor goroutines that are listening on it.

# Panics

This function considers invalid request URLs as fatal. To avoid this make sure
the parameters to Client's constructor are valid.
*/
func (c *Client) getUpdates(ctx context.Context, updateQueue chan<- UpdateProcessor) {
	c.wg.Add(1)
	defer c.wg.Done()
	defer close(updateQueue)

	query := url.Values{} // Stores the update id offset, so should not reset between iterations

	query.Add("limit", "100") // How many updates can all goroutines handle
	query.Add("timeout", "5") // How long the network request should pend for before returning an empty update list

	log.Println("Telegram bot processor started")

	const retry = 10

	failures := 0
	for failures < retry {
		select {
		case <-ctx.Done():
			return
		default:
			req := c.mustGetRequest(ctx, "getUpdates", query, nil)

			result, err := doAPIRequest[[]update](c, req, "/getUpdates")
			if err != nil {
				failures++
				log.Printf("/getUpdates failure #%d: %s\n", failures, err)

				continue
			}

			failures = 0

			log.Println("/getUpdates failure count reset to 0")

			for i, upd := range *result {
				log.Printf("Sending update #%d to the queue", upd.ID)

				// upd is not ok here because of https://stacko6verflow.com/questions/62446118/implicit-memory-aliasing-in-for-loop
				updateQueue <- &(*result)[i]
			}

			// Prevents new updates from being the same thing
			if len(*result) >= 1 {
				lastUpdate := (*result)[len(*result)-1]
				query.Set("offset", strconv.FormatInt(lastUpdate.ID+1, 10))
			}
		}
	}
	panic("Bot encountered too many errors while interacting with Telegram API")
}

// Combines an update with conversation state
func (c *Client) stateQueue(updateCh chan UpdateProcessor, stateCh chan<- updateHandler) {
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
processUpdates method should be run in a goroutine and will process updates
that come through the channel.

Stop this goroutine by closing the channel
*/
func (c *Client) processUpdates(updateQueue <-chan updateHandler) {
	c.wg.Add(1)
	defer c.wg.Done()

	for update := range updateQueue {
		state, actions := update.processor.processTelegramUpdate(update.state)
		for _, action := range actions {
			endpoint, query := action.telegramBotAction()

			req := c.mustGetRequest(context.Background(), endpoint, query, nil)

			_, err := doAPIRequest[struct{}](c, req, "/"+endpoint)
			if err != nil {
				log.Printf("Error while performing bot action: %s\n", err)
			}
		}

		c.releaseState(state)
	}
}

// Makes a GET method request from components. Panics if the request cannot be created
func (c *Client) mustGetRequest(ctx context.Context, endpoint string, query url.Values, body io.Reader) *http.Request {
	url := url.URL{ //nolint:exhaustivestruct,exhaustruct // Zero-values are okay here
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

//nolint:dupword
/*
doApiRequest performs the Telegram API request, does error handling, wraps errors
and then returns the payload of the request

`logID` is the string that will be in the logs and error messages and is used
to track what method failed. Can be arbitrary.

# Panics

Panics if the status code is 401 Unauthorized which is usually caused by an invalid bot token.

Or if the telegram API sends a "Migrate to chat ID" error, which is unsupported as of now and could cause weird bugs.

Will also panic if the telegram API returns an error many timess in a row (probably 10 times,
but look at the source to see the exact value.
*/
func doAPIRequest[T any](c *Client, req *http.Request, logID string) (*T, error) {
	const retry = 3
	for i := 0; i < retry; i++ {
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("network error: (%s): %w", logID, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return nil, fmt.Errorf("could not read response body: (%s): %w", logID, err)
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
			return nil, fmt.Errorf("parsing %s json response error: %w", logID, err)
		}

		if !data.Ok {
			err = apiError{What: logID, ErrorCode: data.ErrorCode, Description: data.Description}
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

	return nil, retryError{What: logID, Retries: retry}
}

type apiError struct {
	What        string // What action caused the error
	ErrorCode   int    // Error code (usually can be interpreted as HTTP response codes)
	Description string // Human readable explanation of the status code
}

func (e apiError) Error() string {
	return fmt.Sprintf("telegram API error (%s), %d: %q", e.What, e.ErrorCode, e.Description)
}

type retryError struct {
	What    string // What action was done `Retries` times and failed.
	Retries int    // How many times the action was performed before giving up
}

func (e retryError) Error() string {
	return fmt.Sprintf("telegram API error (%s), retry limit (%d) exceeded: ", e.What, e.Retries)
}
