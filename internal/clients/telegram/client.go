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

type Client struct {
	host           string
	basePath       string
	client         http.Client
	wg             sync.WaitGroup
	stopProcessing context.CancelFunc
}

func NewClient(host, token string) Client {
	return Client{
		host:     host,
		basePath: "bot" + token,
		client:   http.Client{},
	}
}

func (c *Client) Start(goroutines uint) {
	if goroutines < 1 {
		panic(fmt.Sprintf("Should never start a telegram bot with less than 1 processor threads. Was asked to use %d threads.", goroutines))
	}
	var (
		updateQueue = make(chan UpdateProcessor, goroutines)
		ctx, cancel = context.WithCancel(context.Background())
	)
	c.stopProcessing = cancel

	go c.getUpdates(ctx.Done(), updateQueue)
	for i := uint(0); i < goroutines; i++ {
		go c.processUpdates(updateQueue)
	}
}

func (c *Client) Stop() {
	c.stopProcessing()
	c.wg.Wait()
}

/*
getUpdates method should be run in a goroutine and will call getUpdates
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

	const cycleTime = 5 * time.Second // Minimum time between fetching updates
	query.Add("limit", "100")         // How many updates can all goroutines handle
	query.Add("timeout", "10")        // How long the network request should pend for before returning an empty update list

	log.Println("Telegram bot processor started")
	for {
		select {
		case <-stopCh:
			return
		default:
			startTime := time.Now() // Used for rate limiting network requests

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
				log.Fatal("Request should always be valid, %w", err)
			}
			resp, err := c.client.Do(req)
			if err != nil {
				log.Printf("Network error: (%s): %s", c.host, err)
			}
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

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				log.Printf("Could not read responce body: %s", err)
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
					log.Fatalf("API asked to migrate to chat ID %d, which is an unsupported operation", data.Parameters.MigrateToChatId)
				}
				continue
			}

			for _, upd := range data.Result {
				log.Printf("Sending update #%d to the queue", upd.ID)
				updateCopy := upd
				updateQueue <- &updateCopy
			}
			if len(data.Result) >= 1 {
				last_update := data.Result[len(data.Result)-1]
				query.Set("offset", strconv.FormatInt(last_update.ID+1, 10))
			}

			dtFetching := time.Since(startTime) // How long did it take to make a network request and queue updates
			time.Sleep(cycleTime - dtFetching)  // If the difference is negative, continue. Otherwise dont spam the API
		}
	}
}

/*
processUpdates method should be run in a goroutine and will process updates
that come through the channel.

Cancel this goroutine by closing the channel
*/
func (c *Client) processUpdates(updateQueue <-chan UpdateProcessor) {
	c.wg.Add(1)
	defer c.wg.Done()

	for update := range updateQueue {
		for _, action := range update.processTelegramUpdate() {
			endpoint, query := action.telegramBotAction()

			u := url.URL{
				Scheme:   "https",
				Host:     c.host,
				Path:     path.Join(c.basePath, endpoint),
				RawQuery: query.Encode(),
			}
			req, err := http.NewRequest(http.MethodGet, u.String(), nil)
			if err != nil {
				// panic because a request should always be valid
				log.Fatal("Request should always be valid, %w", err)
			}
			resp, err := c.client.Do(req) // ignoring the responce for now
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
					log.Printf("An error occured while performing /%s%s: (%d) %s", endpoint, query.Encode(), isOk.ErrorCode, isOk.Description)
				}
			}
		}
	}
}
