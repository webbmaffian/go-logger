package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/webbmaffian/go-logger/example3"
	"github.com/webbmaffian/go-logger/peer"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var certs example3.Certs

	if err := certs.LoadOrCreate("../certs", "localhost"); err != nil {
		return
	}

	startServer(ctx, &certs)
}

func startServer(ctx context.Context, certs *example3.Certs) (err error) {
	log.Println("starting server")

	if _, err = peer.NewTlsServer(ctx, peer.TlsServerOptions{
		Address:     "localhost:4610",
		PrivateKey:  certs.ServerKey,
		Certificate: certs.ServerCert,
		RootCa:      certs.RootCa,
		EntryProc:   &entryEchoer{},
		ErrorHandler: func(err error) {
			os.Stdout.WriteString("\n")
			log.Println("server:", err)
		},
		Debug: func(msg string) {
			os.Stdout.WriteString("\n")
			log.Println("server:", msg)
		},
	}); err != nil {
		return
	}

	<-ctx.Done()

	return
}
