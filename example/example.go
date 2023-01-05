package main

import (
	"context"
	"crypto/x509/pkix"
	"encoding/binary"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

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
	if err := start(); err != nil {
		log.Fatal(err)
	}
}

func start() (err error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var (
		csr        auth.Csr
		rootCa     auth.Certificate
		serverKey  auth.PrivateKey
		serverCert auth.Certificate
		clientKey  auth.PrivateKey
		clientCert auth.Certificate
	)

	if serverKey, err = auth.CreatePrivateKey(); err != nil {
		return
	}

	if rootCa, err = auth.CreateCertificate(serverKey, nil, auth.CertificateOptions{
		Subject: pkix.Name{
			CommonName: "Log Stream",
		},
		Expiry: time.Now().AddDate(100, 0, 0),
		Type:   auth.Root,
	}); err != nil {
		return
	}

	if serverCert, err = auth.CreateCertificate(serverKey, rootCa, auth.CertificateOptions{
		PublicKey:   serverKey.Public(),
		Expiry:      time.Now().AddDate(100, 0, 0),
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		Type:        auth.Server,
	}); err != nil {
		return
	}

	if clientKey, err = auth.CreatePrivateKey(); err != nil {
		return
	}

	if csr, err = auth.CreateCsr(clientKey); err != nil {
		return
	}

	if clientCert, err = auth.CreateCertificate(serverKey, rootCa, csr, auth.CertificateOptions{
		SubjectKeyId: binary.BigEndian.AppendUint32(nil, 1),
		Expiry:       time.Now().AddDate(100, 0, 0),
		Type:         auth.Client,
	}); err != nil {
		return
	}

	_ = serverCert
	_ = clientCert

	// log.Println("Created cert:\n", cert)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		server := remote.NewServer(remote.ServerOptions{
			Host:        "127.0.0.1",
			Port:        4610,
			RootCa:      rootCa,
			Certificate: serverCert,
			PrivateKey:  serverKey,
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
			Certificate: clientCert,
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
	return
}
