package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/peer"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var certs Certs

	if err := certs.LoadOrCreate("certs", "localhost"); err != nil {
		return
	}

	startServer(ctx, &certs)

	if err := startClient(ctx, &certs); err != nil {
		log.Fatal(err)
	}
}

func startClient(ctx context.Context, certs *Certs) (err error) {
	var (
		pool   *logger.Pool
		client logger.Client
	)

	log.Println("starting client")

	if client, err = peer.NewTlsClient(ctx, peer.TlsClientOptions{
		Address:     "localhost:4610",
		PrivateKey:  certs.ClientKey,
		Certificate: certs.ClientCert,
		RootCa:      certs.RootCa,
		ErrorHandler: func(err error) {
			log.Println("client:", err)
		},
		Debug: func(msg string) {
			log.Println("client:", msg)
		},
	}); err != nil {
		return
	}

	if pool, err = logger.NewPool(client); err != nil {
		return
	}

	log.Println("all set up")
	l := pool.Logger()

	// log.Println("waiting 3 seconds")
	// time.Sleep(time.Second * 3)

	for i := 0; i < 100; i++ {
		if err = ctx.Err(); err != nil {
			return
		}

		log.Println(l.Debug("msg "+strconv.Itoa(i)).Send(), "- WRITTEN")
		// time.Sleep(time.Second)
	}

	log.Println("done")

	log.Println("waiting 3 seconds")
	time.Sleep(time.Second * 3)

	return
}

func startServer(ctx context.Context, certs *Certs) (err error) {
	var (
		server *peer.TlsServer
	)

	log.Println("starting server")

	if server, err = peer.NewTlsServer(ctx, peer.TlsServerOptions{
		Address:     "localhost:4610",
		PrivateKey:  certs.ServerKey,
		Certificate: certs.ServerCert,
		RootCa:      certs.RootCa,
		ErrorHandler: func(err error) {
			log.Println("server:", err)
		},
		Debug: func(msg string) {
			log.Println("server:", msg)
		},
	}); err != nil {
		return
	}

	_ = server

	return
}
