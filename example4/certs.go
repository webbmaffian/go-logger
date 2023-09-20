package example3

import (
	"crypto/x509/pkix"
	"errors"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/webbmaffian/go-logger/auth"
)

type Certs struct {
	RootKey    auth.PrivateKey
	RootCa     auth.Certificate
	ServerKey  auth.PrivateKey
	ServerCert auth.Certificate
	ClientKey  auth.PrivateKey
	ClientCert auth.Certificate
}

func (c *Certs) LoadOrCreate(dir string, domain string) (err error) {
	var (
		pathRootKey    = filepath.Join(dir, "root.key")
		pathRootCa     = filepath.Join(dir, "root.pem")
		pathServerKey  = filepath.Join(dir, "server.key")
		pathServerCert = filepath.Join(dir, "server.pem")
		pathClientKey  = filepath.Join(dir, "client.key")
		pathClientCert = filepath.Join(dir, "client.pem")
	)

	log.Println("loading certs for domain", domain)

	if err = SetPrivFilePerm(dir, true); err != nil {
		if os.IsNotExist(err) {
			log.Println("certs directory doesn't exist - creating")

			if err = os.Mkdir(dir, PermPrivDir); err != nil {
				return
			}
		}
	}

	dirInfo, err := os.Stat(dir)

	if err != nil {
		return
	}

	if !dirInfo.IsDir() {
		return errors.New("certs: invalid directory")
	}

	// Root key
	if err = SetPrivFilePerm(pathRootKey); err != nil {
		if os.IsNotExist(err) {
			log.Println("root key doesn't exist - creating")
			c.RootKey, err = auth.CreatePrivateKey()

			if err != nil {
				return
			}

			err = c.RootKey.ToFile(pathRootKey)
		}
	} else {
		err = c.RootKey.FromFile(pathRootKey)
	}

	if err != nil {
		return
	}

	// Root cert (CA)
	if err = SetPrivFilePerm(pathRootCa); err != nil {
		if os.IsNotExist(err) {
			log.Println("root CA cert doesn't exist - creating")
			c.RootCa, err = auth.CreateCertificate(c.RootKey, nil, auth.CertificateOptions{
				PublicKey: c.RootKey.Public(),
				Subject: pkix.Name{
					Organization:       []string{"The Web Mafia Ltd."},
					OrganizationalUnit: []string{"Log"},
					CommonName:         "webmafia.com",
				},
				Type: auth.Root,
			})

			if err != nil {
				return
			}

			err = c.RootCa.ToFile(pathRootCa)
		}
	} else {
		err = c.RootCa.FromFile(pathRootCa)
	}

	if err != nil {
		return
	}

	// Server key
	if err = SetPrivFilePerm(pathServerKey); err != nil {
		if os.IsNotExist(err) {
			log.Println("server key doesn't exist - creating")
			c.ServerKey, err = auth.CreatePrivateKey()

			if err != nil {
				return
			}

			err = c.ServerKey.ToFile(pathServerKey)
		}
	} else {
		err = c.ServerKey.FromFile(pathServerKey)
	}

	if err != nil {
		return
	}

	// Server cert
	if err = SetPrivFilePerm(pathServerCert); err != nil {
		if os.IsNotExist(err) {
			log.Println("server cert doesn't exist - creating")
			c.ServerCert, err = auth.CreateCertificate(c.RootKey, c.RootCa, auth.CertificateOptions{
				PublicKey:   c.ServerKey.Public(),
				DNSNames:    []string{domain},
				IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
				Type:        auth.Server,
			})

			if err != nil {
				return
			}

			err = c.ServerCert.ToFile(pathServerCert)
		}
	} else {
		err = c.ServerCert.FromFile(pathServerCert)
	}

	if err != nil {
		return
	}

	// Client key
	if err = SetPrivFilePerm(pathClientKey); err != nil {
		if os.IsNotExist(err) {
			log.Println("client key doesn't exist - creating")
			c.ClientKey, err = auth.CreatePrivateKey()

			if err != nil {
				return
			}

			err = c.ClientKey.ToFile(pathClientKey)
		}
	} else {
		err = c.ClientKey.FromFile(pathClientKey)
	}

	if err != nil {
		return
	}

	// Client cert
	if err = SetPrivFilePerm(pathClientCert); err != nil {
		if os.IsNotExist(err) {
			log.Println("client cert doesn't exist - creating")
			c.ClientCert, err = auth.CreateCertificate(c.RootKey, c.RootCa, auth.CertificateOptions{
				BucketIds: []uint32{123},
				PublicKey: c.ClientKey.Public(),
				Type:      auth.Client,
			})

			if err != nil {
				return
			}

			err = c.ClientCert.ToFile(pathClientCert)
		}
	} else {
		err = c.ClientCert.FromFile(pathClientCert)
	}

	return
}
