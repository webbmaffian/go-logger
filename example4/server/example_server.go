package main

import (
	"context"
	"os"
	"os/signal"
	"time"

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
	// log.Println("starting server")

	if _, err = peer.NewTlsServer(ctx, peer.TlsServerOptions{
		Address:       "127.0.0.1:4610",
		PrivateKey:    certs.ServerKey,
		Certificate:   certs.ServerCert,
		RootCa:        certs.RootCa,
		EntryProc:     &entryEchoer{},
		Debug:         peer.DebuggerStdout(),
		ClientTimeout: time.Second * 10,
	}); err != nil {
		return
	}

	<-ctx.Done()

	return
}
