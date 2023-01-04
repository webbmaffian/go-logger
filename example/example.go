package main

import (
	"context"
	"crypto/x509/pkix"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/webbmaffian/go-logger/transports/remote"
	"github.com/webbmaffian/go-logger/transports/remote/auth"
)

// func main() {
// 	privKey, err := auth.CreatePrivateKey()

// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	log.Print("\n", privKey)

// 	rootCa, err := auth.CreateRootCA(pkix.Name{
// 		Organization:       []string{"Webbmaffian AB"},
// 		OrganizationalUnit: []string{"Log Stream"},
// 	}, privKey, time.Now().AddDate(100, 0, 0))

// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	log.Print("\n", rootCa)

// 	csr, err := auth.CreateCsr(privKey)

// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	log.Print("\n", csr)

// 	cert, err := auth.CreateCertificate(csr, rootCa, privKey, time.Now().AddDate(100, 0, 0))

// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	log.Print("\n", cert)
// }

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var (
		err       error
		serverKey auth.PrivateKey
		clientKey auth.PrivateKey
		csr       auth.Csr
		rootCa    auth.Certificate
		cert      auth.Certificate
	)

	if serverKey, err = auth.CreatePrivateKey(); err != nil {
		return
	}

	if rootCa, err = auth.CreateRootCA(pkix.Name{
		CommonName: "Log Stream",
	}, serverKey, time.Now().AddDate(100, 0, 0)); err != nil {
		return
	}

	if clientKey, err = auth.CreatePrivateKey(); err != nil {
		return
	}

	if csr, err = auth.CreateCsr(clientKey); err != nil {
		return
	}

	if cert, err = auth.CreateCertificate(uuid.New(), csr, rootCa, serverKey, time.Now().AddDate(100, 0, 0)); err != nil {
		return
	}

	log.Println("Created cert:\n", cert)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		server := remote.NewServer(remote.ServerOptions{
			Host:       "127.0.0.1",
			Port:       4610,
			RootCa:     rootCa,
			PrivateKey: serverKey,
		})

		if err := server.Listen(ctx); err != nil {
			log.Println(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// time.Sleep(time.Second * 3)

		client := remote.NewClient(ctx, remote.ClientOptions{
			Host:        "127.0.0.1",
			Port:        4610,
			RootCa:      rootCa,
			Certificate: cert,
			PrivateKey:  clientKey,
		})

		for {
			if ctx.Err() != nil {
				break
			}

			client.Write([]byte("hellooo"))
			time.Sleep(time.Second)
		}

	}()

	wg.Wait()
}
