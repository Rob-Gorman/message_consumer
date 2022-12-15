package main

import (
	"delivery/cmd/app"
	"os"
	"os/signal"
	"syscall"

	"context"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		syscall.SIGQUIT,
		os.Interrupt,
	)
	defer cancel()

	app := app.New(ctx)
	app.Run(ctx) // blocking while message channel is open
}
