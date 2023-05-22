package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/auth"
	"github.com/webbmaffian/go-logger/peer"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	if err := client(ctx); err != nil {
		log.Fatal(err)
	}
}

func client(ctx context.Context) (err error) {
	var (
		key  auth.PrivateKey
		cert auth.Certificate
		root auth.Certificate
	)

	if err = key.FromFile("private.key"); err != nil {
		return
	}

	if err = cert.FromFile("certificate.pem"); err != nil {
		return
	}

	if err = root.FromFile("root.pem"); err != nil {
		return
	}

	cli, err := peer.NewTlsClient(ctx, peer.TlsClientOptions{
		Address:     "localhost:4610",
		PrivateKey:  key,
		Certificate: cert,
		RootCa:      root,
	})

	if err != nil {
		return
	}

	pool, err := logger.NewPool(cli)

	if err != nil {
		return
	}

	log := pool.Logger()

	log.Info("hello there %s", "1").Send()
	log.Info("hello there %s", "2").Send()
	log.Info("hello there %s", "3").Send()
	log.Info("hello there %s", "4").Send()
	fmt.Println("sent")

	time.Sleep(time.Second * 3)

	return
}
