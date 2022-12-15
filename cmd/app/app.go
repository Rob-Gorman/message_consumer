package app

import (
	"context"
	"delivery/consumer"
	"delivery/consumer/grpcclient"
	"delivery/consumer/redis"
	"delivery/logger"
	"delivery/types"
	"net/http"
	"runtime"
	"sync"
	"time"
)

var (
	queue       = "grpc"
	workerCount = runtime.NumCPU() // reasonable starting point IMO
	httpTimeout = 5000             // milliseconds
)

type app struct {
	L           logger.AppLogger
	q           consumer.MessageConsumer
	cl          *http.Client
	WorkerCount int
}

func New(ctx context.Context) *app {
	logger := logger.New()
	return &app{
		L:           logger,
		q:           messageConsumer(ctx, queue, logger),
		cl:          &http.Client{Timeout: time.Duration(httpTimeout) * time.Millisecond},
		WorkerCount: workerCount,
	}
}

// Run is the entry point to the Delivery Agent.
// Run is blocking until shutdown, or the message channel is otherwise closed
func (a *app) Run(ctx context.Context) {
	ch := a.q.NewMessageChannel(ctx)
	ack := make(chan interface{}) // initialize chan for ack id's
	defer close(ack)

	a.q.AckMessages(ctx, ack)

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
				a.L.Error("failed to process message request", err.Error())
				// message simply unprocessed and unacked - questionable
				return
			}

			ack <- m.AckID   // ack message
			a.L.Entry(entry) // delivery log
		}(msg)
	}

	wg.Wait() // only blocks after ch closed
}

// instantiates implementation of `consumer.MessageConsumer`` specified by q parameter
func messageConsumer(ctx context.Context, q string, l logger.AppLogger) consumer.MessageConsumer {
	switch q {
	case "redis":
		return redis.New(l)
	case "grpc":
		return grpcclient.New(ctx, l)
	}
	return grpcclient.New(ctx, l)
}
