# Queue Worker

This is a message consumer for messages that indicate endpoints to which to make requests. 

The queue implementation must satisfy the `MessageConsumer` interface:

```go
// MessageConsumer is the queue subscriber shape the Delivery Agent app
// uses to read from the queue. All queue implementations must implement
type MessageConsumer interface {
	// NewMessageChannel is the entry point to the queue
	// it is responsible for establishing a means of communication to the queue
	// and returning a live channel of messages from the queue as type *types.Message
	NewMessageChannel(context.Context) <-chan *types.Message

	// AckMessageChannel is responsible for acking processed messages from the queue to
	// avoid duplication (when the queue implementation supports it)
	// type interface{} to be queue-agnostic
	AckMessages(context.Context, <-chan interface{}) error
}
```

`app.New(queue, logger)` instantiates a specific queue implementation according to a config variable (hardcoded here for convenience)

`app.Run()` reads from the queue and concurrently processes multiple requests.

Because HTTP requests are mostly just waiting, fanning out the processes to multiple workers makes sense. `WorkerCount` is how we bound how many concurrent goroutines we're spinning up. The variable is currently hardcoded to `2 * runtime.NumCPU()` as a starting point, but is probably a good place to begin tweaking performance. I'd guess more would be better since the process is mostly idling.

```go
sem := make(chan struct{}, a.WorkerCount) // buffered to limit goroutines
defer close(sem)

for msg := range ch {
  sem <- struct{}{}
  wg.Add(1)
  go func(m *types.Message) {
    defer func() { <-sem }() // permit next goroutine
    defer wg.Done()

    entry, err := a.makeRequest(ctx, m)
    if err != nil {
      a.L.Error("failed to process message request", err.Error())
      return
    }

    ack <- m.AckID   // ack message
    a.L.Entry(entry) // delivery log
  }(msg)
}
```

The `consumer` package holds the specific queue implementation packages. Currently, it's using the grpc implementation defined in `grpcclient`

Currently our delivery agent application's _Consumer_ id is hardcoded (for convenience), but programmatically generating a new one for every instance makes sense and allows us to easily scale out how many workers are processing from the queue.

Graceful shutdown is handled by way of `signal.NotifyContext`, which trickles down to all streams and closes connections and channels accordingly. The `context` package is miraculous, honestly.