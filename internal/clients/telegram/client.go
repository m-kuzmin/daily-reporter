package telegram

import (
	"context"
	"net/http"
	"sync"
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

func (c *Client) Start(goroutines int) {
	var (
		updateQueue = make(chan telegramUpdateProcessor, goroutines)
		ctx, cancel = context.WithCancel(context.Background())
	)
	c.stopProcessing = cancel

	go c.getUpdates(ctx.Done(), updateQueue)
	for i := 0; i < goroutines; i++ {
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

Stop this goroutine by closing the stopCh.
*/
func (c *Client) getUpdates(stopCh <-chan struct{}, updateQueue chan<- telegramUpdateProcessor) {
	c.wg.Add(1)
	defer c.wg.Done()

	// first call to endpoint should get the current update id and subseq calls should use that id
	for {
		select {
		case <-stopCh:
			close(updateQueue)
			return
		default:
			// Fetch the update
			updateQueue <- update
		}
	}
}

/*
processUpdates method should be run in a goroutine and will process updates
that come through the channel.

Cancel this goroutine by closing the channel
*/
func (c *Client) processUpdates(updateQueue <-chan telegramUpdateProcessor) {
	c.wg.Add(1)
	defer c.wg.Done()

	for update := range updateQueue {
		// Process the update
	}
}
