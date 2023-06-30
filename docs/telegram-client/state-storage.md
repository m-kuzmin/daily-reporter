# State storage for conversation state

In a conversation with a bot the user may use a command with many steps (states) and side effects. If you tell the bot
your name and then quickly send a different message that other message has a chance to be treated as your name. One
solution (and a valid one at that) is to use one thread. However there are commands the bot has that may require some
idle time. An example is requesting data over an API. This idle time can be used by other goroutines to process their
message. But if they do not cooperate and are unaware of each other's "intentions" the UX will degrade. The details are
explained in [README.md](https://github.com/m-kuzmin/daily-reporter/blob/main/docs/telegram-client/README.md).

The solution to synchronizing threads is to use `borrowonce.Storage`. This struct's `Borrow` method returns a `Future`
which you then `Wait` for. When `Wait` returns it gives you the latest value for the key you requested. After you are
finished using the value you have to `Return` it to the `Storage` which then makes another `Wait` return. An important
property of `Futures` Given by `Borrow` is that they are resolved in the same order they were requested.

```go
futureOne, exists := storage.Borrow(key)
futureTwo, exists := storage.Borrow(key)
// futureThree
// ...

value1 := futureOne.Wait()
value2 := futureTwo.Wait() // is blocked until Release is called

storage.Return(key, value) // unblocks futureTwo

// In reality this exact setup would block on line `value2 := wait()`
```
