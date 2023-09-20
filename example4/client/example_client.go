package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/webbmaffian/go-logger"
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

	startClient(ctx, &certs)
}

func startClient(ctx context.Context, certs *example3.Certs) (err error) {
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
		BufferSize:  100,
		ErrorHandler: func(err error) {
			log.Println("client:", err)
		},
		// Debug: func(msg string) {
		// 	log.Println("client:", msg)
		// },
	}); err != nil {
		return
	}

	if pool, err = logger.NewPool(client); err != nil {
		return
	}

	// log.Println("all set up")
	l := pool.Logger()

	// log.Println("waiting 3 seconds")
	// time.Sleep(time.Second * 3)

	log.Println("waiting 3 seconds")
	time.Sleep(time.Second * 3)

	for i := 0; i < 1000_000; i++ {
		if err = ctx.Err(); err != nil {
			return
		}

		// entry := l.Debug("msg %s", strconv.Itoa(i))
		entry := l.Err("Foobar: %d with 50%% off", "123").Cat(1).Tag("127.0.0.1", "foo@bar.baz", 403).Meta("Specific error", "räksmörgås")
		entry.Send()

		os.Stdout.WriteString(fmt.Sprintf("Sent %4d\r", i+1))
		// time.Sleep(time.Millisecond * 1)

		// log.Println("\n" + example3.FormatEntry(entry, "<"))
		// time.Sleep(time.Second)
	}

	os.Stdout.WriteString("\n")

	log.Println("done")

	log.Println("waiting 3 seconds")
	time.Sleep(time.Second * 3)

	return
}
