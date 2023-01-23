package main

import (
	"context"
	"crypto/x509/pkix"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/auth"
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
	// var trace [16]uintptr
	// n := runtime.Callers(0, trace[:])

	// log.Println(n, trace[:n])

	// frames := runtime.CallersFrames(trace[:n])

	// for {
	// 	frame, ok := frames.Next()

	// 	log.Printf("%+v\n", frame)
	// 	if !ok {
	// 		break
	// 	}
	// }

	// // log.Println(runtime.Caller(0))

	// return

	/*
		TODO:
		- Add bucket IDs (uint32) to Subject Key ID in TLS certificate.
		- Check that the provided bucket ID in each log entry matches a bucket ID in certificate.
	*/
	// log.Printf("%x\n", binary.LittleEndian.Uint32(binary.BigEndian.AppendUint32(nil, 12341234)))
	// return

	// var buf [1024]byte

	// e := logger.Entry{
	// 	id:         xid.New(),
	// 	category:   "foobar",
	// 	procId:     "barfoo",
	// 	message:    "lorem ipsum dolor sit amet",
	// 	tags:       [32]string{"foo", "bar", "baz"},
	// 	tagsCount:  3,
	// 	metaKeys:   [32]string{"foo", "bar", "baz"},
	// 	metaValues: [32]string{"foo", "bar", "baz"},
	// 	metaCount:  3,
	// 	level:      6,
	// }

	// var e2 logger.Entry

	// size := e.Encode(buf[:])

	// for i := 0; i < b.N; i++ {
	// 	e2.Decode(buf[:size])
	// 	log.Printf("%+v\n", e2)
	// 	b.FailNow()
	// }

	// return
	if err := testTlsServerOnly(); err != nil {
		log.Fatal(err)
	}
}

func testTLS() (err error) {
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
		DNSNames:    []string{"localhost"},
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
		SubjectKeyId: 1,
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

		server := testServer(ctx)

		if err := server.Listen(logger.ServerTLS{
			Address:     "127.0.0.1:4610",
			RootCa:      rootCa,
			Certificate: serverCert,
			PrivateKey:  serverKey,
		}); err != nil {
			log.Println(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// time.Sleep(time.Second * 3)

		client := logger.NewClient(ctx, &logger.ClientTLS{
			Address:     "localhost:4610",
			RootCa:      rootCa,
			Certificate: clientCert,
			PrivateKey:  clientKey,
		})

		logger := logger.New(ctx, client)

		var i int

		for {
			if ctx.Err() != nil {
				break
			}

			i++

			logger.Debug("Hi there " + strconv.Itoa(i))

			// client.Write([]byte("hellooo"))
			// time.Sleep(time.Second)
		}

	}()

	wg.Wait()
	return
}

func testTCP() (err error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		server := testServer(ctx)

		if err := server.Listen(logger.ServerTCP{
			Address: "127.0.0.1:4610",
		}); err != nil {
			log.Println(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// time.Sleep(time.Second * 3)

		client := logger.NewClient(ctx, &logger.ClientTCP{
			Address: "localhost:4610",
		})

		logger := logger.New(ctx, client)

		var i int

		for {
			if ctx.Err() != nil {
				break
			}

			i++

			logger.Debug("Hi there " + strconv.Itoa(i))

			// client.Write([]byte("hellooo"))
			// time.Sleep(time.Second)
		}

	}()

	wg.Wait()
	return
}

func testUDP() (err error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		server := testServer(ctx)

		if err := server.Listen(logger.ServerUDP{
			Address: "127.0.0.1:4610",
		}); err != nil {
			log.Println(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// time.Sleep(time.Second * 3)

		client := logger.NewClient(ctx, &logger.ClientUDP{
			Address: "127.0.0.1:4610",
		})

		logger := logger.New(ctx, client)

		var i int

		for {
			if ctx.Err() != nil {
				break
			}

			i++

			logger.Err("Hi there " + strconv.Itoa(i))

			// client.Write([]byte("hellooo"))
			// time.Sleep(time.Second)
		}

	}()

	wg.Wait()
	return
}

func testUnixgram() (err error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		server := testServer(ctx)

		if err := server.Listen(logger.ServerUnixgram{
			Address: "test.socket",
		}); err != nil {
			log.Println(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// time.Sleep(time.Second * 3)

		client := logger.NewClient(ctx, &logger.ClientUnixgram{
			Address: "test.socket",
		})

		logger := logger.New(ctx, client)

		var i int

		for {
			if ctx.Err() != nil {
				break
			}

			i++

			logger.Debug("Hi there " + strconv.Itoa(i))

			// client.Write([]byte("hellooo"))
			// time.Sleep(time.Second)
		}

	}()

	wg.Wait()
	return
}

func testUnix() (err error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		server := testServer(ctx)

		if err := server.Listen(logger.ServerUnix{
			Address: "test.socket",
		}); err != nil {
			log.Println(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// time.Sleep(time.Second * 3)

		client := logger.NewClient(ctx, &logger.ClientUnix{
			Address: "test.socket",
		})

		logger := logger.New(ctx, client)

		var i int

		for {
			if ctx.Err() != nil {
				break
			}

			i++

			logger.Debug("Hi there " + strconv.Itoa(i))

			// client.Write([]byte("hellooo"))
			// time.Sleep(time.Second)
		}

	}()

	wg.Wait()
	return
}

func testServer(ctx context.Context) logger.Server {
	return logger.NewServer(ctx, logger.EntryReaderCallback(func(b []byte) (err error) {
		var e logger.Entry

		// log.Println(b)
		// return

		if err = e.Decode(b); err != nil {
			return
		}

		log.Printf("server: got message: %d: %+v %+v\n", e.MetricCount, e.MetricKeys, e.MetricValues)
		// log.Printf("server: %+v\n", e.StackTracePaths[:e.StackTraceCount])
		// log.Printf("server:  %+v\n", e.StackTraceRowNumbers[:e.StackTraceCount])
		return
	}))
}

func testPipe() (err error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		server := testServer(ctx)

		if err := server.Listen(logger.ServerTCP{
			Address: "127.0.0.1:4610",
		}); err != nil {
			log.Println(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		server := logger.NewServer(ctx, logger.NewClient(ctx, &logger.ClientTCP{
			Address: "127.0.0.1:4610",
		}))

		if err := server.Listen(logger.ServerTCP{
			Address: "127.0.0.1:4609",
		}); err != nil {
			log.Println(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// time.Sleep(time.Second * 3)

		client := logger.NewClient(ctx, &logger.ClientTCP{
			Address: "127.0.0.1:4609",
		})

		logger := logger.New(ctx, client)

		var i int

		for {
			if ctx.Err() != nil {
				break
			}

			i++

			logger.Debug("Hi there " + strconv.Itoa(i))

			// client.Write([]byte("hellooo"))
			time.Sleep(time.Second)
		}

	}()

	wg.Wait()
	return
}

func testServerOnly() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	server := testServer(ctx)
	if err := server.Listen(logger.ServerTCP{
		Address: "127.0.0.1:4610",
	}); err != nil {
		log.Fatal(err)
	}

	return nil
}

func testTlsServerOnly() (err error) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	var (
		// csr        auth.Csr
		rootCa     auth.Certificate
		serverKey  auth.PrivateKey
		serverCert auth.Certificate
		// clientKey  auth.PrivateKey
		// clientCert auth.Certificate
	)

	// if serverKey, err = auth.CreatePrivateKey(); err != nil {
	// 	return
	// }

	// if rootCa, err = auth.CreateCertificate(serverKey, nil, auth.CertificateOptions{
	// 	Subject: pkix.Name{
	// 		CommonName: "Log Stream",
	// 	},
	// 	Expiry: time.Now().AddDate(100, 0, 0),
	// 	Type:   auth.Root,
	// }); err != nil {
	// 	return
	// }

	// if serverCert, err = auth.CreateCertificate(serverKey, rootCa, auth.CertificateOptions{
	// 	PublicKey:   serverKey.Public(),
	// 	Expiry:      time.Now().AddDate(100, 0, 0),
	// 	DNSNames:    []string{"localhost"},
	// 	IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	// 	Type:        auth.Server,
	// }); err != nil {
	// 	return
	// }

	// if clientKey, err = auth.CreatePrivateKey(); err != nil {
	// 	return
	// }

	// if csr, err = auth.CreateCsr(clientKey); err != nil {
	// 	return
	// }

	// if clientCert, err = auth.CreateCertificate(serverKey, rootCa, csr, auth.CertificateOptions{
	// 	SubjectKeyId: 1,
	// 	Expiry:       time.Now().AddDate(100, 0, 0),
	// 	Type:         auth.Client,
	// }); err != nil {
	// 	return
	// }

	if err = rootCa.FromFile("root.pem"); err != nil {
		log.Fatal(err)
	}

	if err = serverKey.FromFile("server.key"); err != nil {
		log.Fatal(err)
	}

	if err = serverCert.FromFile("server.cert"); err != nil {
		log.Fatal(err)
	}

	// rootCa.ToFile("root.pem")
	// clientKey.ToFile("client.key")
	// clientCert.ToFile("client.cert")
	// serverKey.ToFile("server.key")
	// serverCert.ToFile("server.cert")

	// log.Println(rootCa)
	// log.Println(clientKey)
	// log.Println(clientCert)

	server := testServer(ctx)
	if err := server.Listen(logger.ServerTLS{
		Address:     "127.0.0.1:4610",
		RootCa:      rootCa,
		Certificate: serverCert,
		PrivateKey:  serverKey,
	}); err != nil {
		log.Println(err)
	}

	return nil
}
