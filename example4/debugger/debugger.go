package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/webbmaffian/go-logger/debug"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	addr := "localhost:1234"

	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	log.Println("Listening on", addr)

	if err := debug.Read(ctx, addr, os.Stdout); err != nil {
		log.Println(err)
	}
}
