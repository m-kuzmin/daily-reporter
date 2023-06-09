# Telegram client inner workings

# Problem with sequential processing

If 2 groups send a message at the same time, a message that could have been processed quickly (e.g. /start) will
have to wait in a queue. Instead we can process all messages in parallel. This way if one message is waiting for
a responce from a remote API, the other messages can be processed.

![images/README.md/image-1](https://user-images.githubusercontent.com/71077087/244382135-905e7c95-ebe1-40eb-9b60-f10620e3b1b1.png)

Instead of `go processUpdate(update)` for every message we have a fixed number of goroutines that look something
like this:
```go
goroutines := 10
updateCh := make(chan update, goroutines)

// spin up `goroutines` of these functions so they get and process bot
// messages in parallel. The parallelism comes from the fact you made N
// copies of this funcion, not 10 loop iterations inside them.
for (i := 0; i < goroutines; i++) {

    go /*telegram.Client.processUpdates*/ func(updateCh <-chan update) {
        for update := range updateCh {
            // respond to the message or something
        }
    }(updateCh)

}
```
The messages come in through the channel and each iteration of the loop processes each update.

The updates themselves are generated inside `getUpdates()` which looks something like this:
```go
ctx := context.Background()

go /*telegram.Client.getUpdates*/ func(stopCh <-chan struct{}, updateCh <-chan update) {
    for {
        // check if stopCh is closed
        
        // get the messages from API
        
        // send them to the updateCh
    }
}(ctx.Done(), updateCh) // same as in the code block above
```

After a user's message is in the channel, whichever goroutine is trying to read from the channel will receive
that message.

# Problem with parallel processing

Some messages are special. They are special because the bot asks a question (e.g. what is your api token) in responce 
to a command. The next message the users sends has to be treated as a responce to the bot's question. This special
treatment of messages is achieved by storing conversation state.

This state is just a struct that implements an interface for processing messages.

```go
type ConversationStateHandler interface {
    telegramMessage(message) (ConversationStateHandler, []telegramBotActor)
    
    // ... other methods to process different kinds of updates like buttons.
}
```

This interface is the thing that does the actual work inside of `processUpdates()`. 

To change the converstion's state a method returns a different struct that also implements
`ConversationStateHandler`. The second return parameter is the actions the bot should make like send a
message or do something else.

![images/README.md/image-2](https://user-images.githubusercontent.com/71077087/244382154-c7b44565-ea88-4f12-b11c-827359485201.png)

The only issue is that if you have 2 `ConversationStateHandler`s that process the same conversation this will
happen:

![images/README.md/image-3](https://user-images.githubusercontent.com/71077087/244382167-c19ce93f-5440-46cc-9ad6-45e6c5037f15.png)

- First handler will change the conversation state to `A`.
- Second handler will change the conversation state to `B` (overwrites `A`).

Now the user is confused. Because state `A` may have required the user to answer a question, but state `B` (current
state) ignores these answers.

The state has been overwritten due to a race condition. And this race condition would occur if the user issues
two commands at the same time.

# Solution (get multithreading, but without the race conditions)

![images/README.md/image-4](https://user-images.githubusercontent.com/71077087/244382183-71fd6909-47ea-4150-b34d-c3baaf55c656.png)

To prevent 2 updates using the same state at the same time you have to make sure they run one after the other. So
if the user sent the following:

1.
```text
/addApiKey
```
2.
```text
/start
```

1. The first message will look at the current state.
2. The state is `Root`
3. The first message takes that root state and gets sent into one of `telegram.Client.processUpdates()`
4. Inside the goroutine the state is updated to `AddApiKey`
-----------
5. The second message tries to take the state.
6. It fails to take the state because its "lent out", or taken by `/addApiKey`
7. Second message waits for `/addApiKey` to release the state.
8. Once released it then takes the new state (not `Root`, because now it is `AddApiKey`)
9. The second message gets sent to processing as in step 3.
