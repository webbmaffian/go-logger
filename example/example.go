package main

import (
	"crypto/x509/pkix"
	"log"
	"time"

	"github.com/webbmaffian/go-logger/auth"
)

func main() {
	privKey, err := auth.CreatePrivateKey()

	if err != nil {
		log.Fatal(err)
	}

	log.Print("\n", privKey)

	rootCa, err := auth.CreateRootCA(pkix.Name{
		Organization:       []string{"Webbmaffian AB"},
		OrganizationalUnit: []string{"Log Stream"},
	}, privKey, time.Now().AddDate(100, 0, 0))

	if err != nil {
		log.Fatal(err)
	}

	log.Print("\n", rootCa)

	csr, err := auth.CreateCsr(privKey)

	if err != nil {
		log.Fatal(err)
	}

	log.Print("\n", csr)

	cert, err := auth.CreateCertificate(csr, rootCa, privKey, time.Now().AddDate(100, 0, 0))

	if err != nil {
		log.Fatal(err)
	}

	log.Print("\n", cert)
}
