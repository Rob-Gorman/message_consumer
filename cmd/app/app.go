package app

import (
	"context"
	"delivery/internal/consumer"
	"delivery/internal/logger"
	"delivery/internal/types"
	"net/http"
	"runtime"
	"sync"
	"time"
)

var (
	workerCount = 2 * runtime.NumCPU() // reasonable starting point IMO
	httpTimeout = 5000                 // milliseconds
)

type app struct {
	Log         logger.AppLogger
	queue       consumer.MessageConsumer
	client      *http.Client
	WorkerCount int
}

// factory function for App instance
func New(l logger.AppLogger, q consumer.MessageConsumer) *app {
	return &app{
		Log:         l,
		queue:       q,
		client:      &http.Client{Timeout: time.Duration(httpTimeout) * time.Millisecond},
		WorkerCount: workerCount,
	}
}

// Run is the entry point to the Delivery Agent.
// Run is blocking until shutdown, or the message channel is otherwise closed
func (a *app) Run(ctx context.Context) {
	ch := a.queue.NewMessageChannel(ctx)
	ack := make(chan interface{}) // initialize chan for ack id's
	defer close(ack)

	a.queue.AckMessages(ctx, ack)

	// ensures goroutines can complete when `ch` closes
	var wg sync.WaitGroup

	// blocking channel to limit number of goroutines
	sem := make(chan struct{}, a.WorkerCount)
	defer close(sem)

	for msg := range ch {
		sem <- struct{}{}
		wg.Add(1)
		go func(m *types.Message) {
			defer func() { <-sem }()
			defer wg.Done()

			entry, err := a.makeRequest(ctx, m)
			if err != nil {
				a.Log.Error("failed to process message request", err.Error())
				// message simply unprocessed and unacked - questionable
				return
			}

			ack <- m.AckID     // ack message
			a.Log.Entry(entry) // delivery log
		}(msg)
	}

	wg.Wait() // only blocks after ch closed
}
