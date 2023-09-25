package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/rs/xid"
	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/example3"
	"github.com/webbmaffian/go-logger/peer"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var certs example3.Certs

	if err := certs.LoadOrCreate("../certs", "localhost"); err != nil {
		log.Fatal(err)
	}

	if err := startClient(ctx, &certs); err != nil {
		log.Fatal(err)
	}

	// if err := startLiveClient(ctx); err != nil {
	// 	log.Fatal(err)
	// }
}

func startClient(ctx context.Context, certs *example3.Certs) (err error) {
	var (
		pool   *logger.Pool
		client *peer.TlsClient
	)

	log.Println("starting client")

	if client, err = peer.NewTlsClient(peer.TlsClientOptions{
		Address:     "127.0.0.1:4610",
		PrivateKey:  certs.ClientKey,
		Certificate: certs.ClientCert,
		RootCa:      certs.RootCa,
		BufferSize:  100,
		WriteMethod: peer.WriteOrBlock,
	}); err != nil {
		return
	}

	// defer client.Close(ctx)

	if pool, err = logger.NewPool(client, logger.PoolOptions{
		// BucketId: 1684512816,
		BucketId: 123,
	}); err != nil {
		return
	}

	// log.Println("all set up")
	l := pool.Logger()

	// log.Println("waiting 3 seconds")
	// time.Sleep(time.Second * 3)

	// log.Println("waiting 3 seconds")
	// time.Sleep(time.Second * 3)

	if true {
		for i := 0; i < 500; i++ {
			if err = ctx.Err(); err != nil {
				return
			}

			// time.Sleep(time.Second * 6)

			entry := l.Debug("msg " + strconv.Itoa(i+1))
			// entry := l.Err("Foobar: %d with 50%% off", "123").Cat(1).Tag("127.0.0.1", "foo@bar.baz", 403).Meta("Specific error", "räksmörgås")
			entry.Send()

			// os.Stdout.WriteString(fmt.Sprintf("Sent %4d\r", i+1))

			// if i == 4 {
			// time.Sleep(time.Second * 6)
			// }

			// log.Println("\n" + example3.FormatEntry(entry, "<"))
			// time.Sleep(time.Second)
		}
	}

	// os.Stdout.WriteString("\n")

	// log.Println("done writing entries")

	// log.Println("waiting 3 seconds")
	// time.Sleep(time.Second * 3)

	// <-ctx.Done()

	return client.CloseGracefully(time.Second * 10)
}

func startLiveClient(ctx context.Context) (err error) {
	var (
		pool   *logger.Pool
		client *peer.TlsClient
		certs  example3.Certs
	)

	if err = certs.RootCa.FromFile("../cert-live/root-ca.pem"); err != nil {
		return
	}

	if err = certs.ClientKey.FromFile("../cert-live/private.key"); err != nil {
		return
	}

	if err = certs.ClientCert.FromFile("../cert-live/certificate.pem"); err != nil {
		return
	}

	log.Println("starting client")

	if client, err = peer.NewTlsClient(peer.TlsClientOptions{
		Address:     "wm.log.center:4610",
		PrivateKey:  certs.ClientKey,
		Certificate: certs.ClientCert,
		RootCa:      certs.RootCa,
		BufferSize:  128,
	}); err != nil {
		return
	}

	if pool, err = logger.NewPool(client, logger.PoolOptions{
		BucketId: 1695284931,
	}); err != nil {
		return
	}

	// log.Println("all set up")
	l := pool.Logger().Cat(1).Tag("127.0.0.1", "foo@bar.baz", xid.New())

	// log.Println("waiting 3 seconds")
	// time.Sleep(time.Second * 3)

	// log.Println("waiting 3 seconds")
	// time.Sleep(time.Second * 3)
	l.Info("First message").Send()
	time.Sleep(time.Second * 70)
	l.Info("Second message").Send()
	time.Sleep(time.Second * 70)
	l.Info("Third message").Send()

	// var wg sync.WaitGroup

	// for worker := 0; worker < 1; worker++ {
	// 	wg.Add(1)
	// 	go func(worker int) {
	// 		for i := 0; i < 100; i++ {
	// 			l.Info("Msg %d from worker %d").Tag(i+1, worker+1, "a", "b", "c", "d", "e").Meta("Specific error", "räksmörgås").Send()
	// 			time.Sleep(time.Millisecond)
	// 		}

	// 		wg.Done()
	// 	}(worker)
	// }

	// for i := 0; i < 10; i++ {
	// 	if err = ctx.Err(); err != nil {
	// 		return
	// 	}

	// 	// entry := l.Debug("msg %s", strconv.Itoa(i))
	// 	// entry :=

	// 	// os.Stdout.WriteString(fmt.Sprintf("Sent %4d\r", i+1))
	// 	// time.Sleep(time.Second * 6)

	// 	// log.Println("\n" + example3.FormatEntry(entry, "<"))
	// 	// time.Sleep(time.Second)
	// }

	// os.Stdout.WriteString("\n")

	// wg.Wait()
	log.Println("done")

	// if tlsClient, ok := client.(*peer.TlsClient); ok {
	// 	log.Println("closing gracefully...")
	// 	// ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	// 	// defer cancel()
	// 	err = tlsClient.Close(ctx)
	// }

	// log.Println("done waiting")

	// log.Println("waiting 3 seconds")
	// time.Sleep(time.Second * 3)

	return client.CloseGracefully(time.Second * 5)
}
