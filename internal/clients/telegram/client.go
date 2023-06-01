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
	wg             sync.WaitGroup     //Used to make sure all processor threads are done
	stopProcessing context.CancelFunc // Triggers the shutdown
}

// Creates a new client.
//
// `host` is the address to the server, without `https://`
//
// `token` is the bot token for the API
func NewClient(host, token string) Client {
	return Client{
		host:     host,
		basePath: "bot" + token,
		client:   http.Client{},
	}
}

/*
Starts the client in the background. It will fetch updates via `Client.getUpdates()`
and sends them to a queue. Queue items are recieved by one of `Client.processUpdates()`
which are going to process all updates in parallel

`goroutines` specifies how many threads to use in parallel.

# Panics

This function considers 0 or less goroutines as fatal.
*/
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

	go c.getUpdates(ctx.Done(), updateQueue)
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
func (c *Client) getUpdates(stopCh <-chan struct{}, updateQueue chan<- UpdateProcessor) {
	c.wg.Add(1)
	defer c.wg.Done()
	defer close(updateQueue)

	query := url.Values{} // Stores the update id offset, so should not reset between iterations

	query.Add("limit", "100") // How many updates can all goroutines handle
	query.Add("timeout", "5") // How long the network request should pend for before returning an empty update list

	log.Println("Telegram bot processor started")
	for {
		select {
		case <-stopCh:
			return
		default:
			req := c.mustGetRequest("getUpdates", query, nil)
			result, err := doApiRequest[[]update](c, req, "/getUpdates")
			if err != nil {
				continue
			}

			for _, upd := range *result {
				log.Printf("Sending update #%d to the queue", upd.ID)
				updateQueue <- &upd
			}

			// Prevents new updates from being the same thing
			if len(*result) >= 1 {
				last_update := (*result)[len(*result)-1]
				query.Set("offset", strconv.FormatInt(last_update.ID+1, 10))
			}
		}
	}
}

// Combines an update with conversation state
func (c *Client) stateQueue(updateCh chan UpdateProcessor, stateCh chan<- updateHandler) {
	c.wg.Add(1)
	defer c.wg.Done()
	// TODO: Hold back updates that could cause race conditions in parallel processing
	defer close(stateCh)
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
func (c *Client) takeState() ConversationStateHandler {
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

			req := c.mustGetRequest(endpoint, query, nil)
			_, err := doApiRequest[struct{}](c, req, "/"+endpoint)
			if err != nil {
				log.Printf("Error while performing bot action: %s", err)
			}
		}
		c.releaseState(state)
	}
}

// Makes a GET method request from components. Panics if the request cannot be created
func (c *Client) mustGetRequest(endpoint string, query url.Values, body io.Reader) *http.Request {
	u := url.URL{
		Scheme:   "https",
		Host:     c.host,
		Path:     path.Join(c.basePath, endpoint),
		RawQuery: query.Encode(),
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), body)
	if err != nil {
		// panic because a request should always be valid
		log.Fatal("Error: Request should always be valid, %w", err)
	}
	return req
}

/*
Performs the Telegram API request, does error handling, wraps errors and then returns the payload of the request

# Panics

Panics if the status code is 401 Unauthorized which is usually caused by an invalid bot token.

Or if the telegram API sends a "Migrate to chat ID" error, which is unsupported as of now and could cause weird bugs.
*/
func doApiRequest[T any](c *Client, req *http.Request, logID string) (_ *T, err error) {
	for i := 0; i < 3; i++ {
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Network error: (%s): %s", logID, err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("Could not read responce body: (%s): %s", logID, err)
		}

		// Parse

		var data struct {
			Ok          bool   `json:"ok"`
			ErrorCode   int    `json:"error_code,omitempty"`
			Description string `json:"description,omitempty"`
			Parameters  struct {
				MigrateToChatId int64         `json:"migrate_to_chat_id,omitempty"`
				RertyAfter      time.Duration `json:"retry_after,omitempty"`
			} `json:"parameters,omitempty"`
			Result T `json:"result"`
		}

		if err = json.Unmarshal(body, &data); err != nil {
			err = fmt.Errorf("Parsing %s json responce error: %s", logID, err)
			continue
		}
		if !data.Ok {
			err = fmt.Errorf("Telegram API error, %d: %q", data.ErrorCode, data.Description)
			log.Println(err)
			if data.ErrorCode == http.StatusUnauthorized {
				log.Fatal("Token is likely invalid")
			}
			if data.Parameters.RertyAfter != 0 {
				time.Sleep(time.Duration(data.Parameters.RertyAfter * time.Second))
				continue
			}
			if data.Parameters.MigrateToChatId != 0 {
				log.Fatalf("API asked to migrate to chat ID %d, which is an unsupported operation",
					data.Parameters.MigrateToChatId)
			}
			continue
		}

		return &data.Result, nil
	}
	return nil, err
}
