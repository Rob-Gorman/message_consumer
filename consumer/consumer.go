package consumer

import (
	"context"
	"delivery/types"
)

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
