package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/kpango/fastime"
	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/auth"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	if err := startClient(ctx); err != nil {
		log.Fatal(err)
	}
}

func startClient(ctx context.Context) (err error) {
	var (
		clientKey  auth.PrivateKey
		clientCert auth.Certificate
		rootCa     auth.Certificate
	)

	if err = clientKey.FromFile("client.key"); err != nil {
		return
	}

	if err = clientCert.FromFile("client.cert"); err != nil {
		return
	}

	if err = rootCa.FromFile("root.pem"); err != nil {
		return
	}

	entryPool := logger.NewEntryPool()
	clock := fastime.New().StartTimerD(ctx, time.Second)
	pool := logger.LoggerPool{
		EntryPool: entryPool,
		EntryProcessor: logger.NewClient(ctx, &logger.ClientTLS{
			Address:     "localhost:4610",
			PrivateKey:  clientKey,
			Certificate: clientCert,
			RootCa:      rootCa,
			Clock:       clock,
		}, entryPool, logger.ClientOptions{
			BufferSize: 8,
		}),
		Clock:    clock,
		BucketId: 1680628490,
	}

	log.Println("all set up")
	l := pool.Logger()

	for i := 0; i < 10; i++ {
		l.Debug("msg " + strconv.Itoa(i)).Send()
	}

	log.Println("done, waiting 1 sec...")
	time.Sleep(time.Second)
	return
}
