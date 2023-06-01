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
			// Fetch the update

			u := url.URL{
				Scheme:   "https",
				Host:     c.host,
				Path:     path.Join(c.basePath, "getUpdates"),
				RawQuery: query.Encode(),
			}
			req, err := http.NewRequest(http.MethodGet, u.String(), nil)
			if err != nil {
				// panic because a request should always be valid
				log.Fatal("Error: Request should always be valid, %w", err)
			}
			resp, err := c.client.Do(req)
			if err != nil {
				log.Printf("Network error: (%s): %s", c.host, err)
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				log.Printf("Could not read responce body: %s", err)
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
				Result []update `json:"result"`
			}

			if err = json.Unmarshal(body, &data); err != nil {
				log.Printf("Parsing /getUpdates json error: %s", err)
			}
			if !data.Ok {
				log.Printf("Telegram API error, %d: %q", data.ErrorCode, data.Description)
				if data.ErrorCode == 401 {
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

			// Queue updates

			for _, upd := range data.Result {
				log.Printf("Sending update #%d to the queue", upd.ID)
				updateQueue <- &upd
			}

			// Prevents new updates from being the same thing
			if len(data.Result) >= 1 {
				last_update := data.Result[len(data.Result)-1]
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

			u := url.URL{
				Scheme:   "https",
				Host:     c.host,
				Path:     path.Join(c.basePath, endpoint),
				RawQuery: query.Encode(),
			}

			// TODO: DRY the request code here, and in `getUpdates`

			req, err := http.NewRequest(http.MethodGet, u.String(), nil)
			if err != nil {
				// panic because a request should always be valid
				log.Fatal("Request should always be valid, %w", err)
			}
			resp, err := c.client.Do(req)
			if err != nil {
				log.Printf("Network error: (%s): %s", c.host, err)
			} else {
				var isOk struct {
					Ok          bool   `json:"ok"`
					ErrorCode   int    `json:"error_code,omitempty"`
					Description string `json:"description,omitempty"`

					Result []message `json:"message"`
				}

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					log.Printf("Could not read responce body: %s", err)
				}
				if err = json.Unmarshal(body, &isOk); err != nil {
					log.Printf("Parsing /%s action responce json error: %s", endpoint, err)
				}

				if !isOk.Ok {
					log.Printf("An error occured while performing /%s?%s: (%d) %s",
						endpoint, query.Encode(), isOk.ErrorCode, isOk.Description)
				}
			}
			c.releaseState(state)
		}
	}
}
