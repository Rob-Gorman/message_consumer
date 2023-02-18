package main

import (
	"delivery/cmd/app"
	"delivery/internal/consumer/grpcclient"
	"delivery/internal/logger"
	"os/signal"
	"syscall"

	"context"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGINT,
	)
	defer cancel()

	l := logger.New()
	gc := grpcclient.New(ctx, l)
	app := app.New(l, gc)

	app.Run(ctx) // blocking while message channel is open
}
