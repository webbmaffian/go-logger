package logger

import (
	"crypto/aes"
	"crypto/cipher"
	"testing"
)

func BenchmarkEncrypt(b *testing.B) {
	var err error

	s := &serverConnection{
		authenticator:  dummyAuthenticator{},
		rawEntryReader: dummyRawEntryReader{},
	}

	aes, err := aes.NewCipher(s.clientSecret[:])

	if err != nil {
		b.Fatal(err)
	}

	if s.encrypt, err = cipher.NewGCM(aes); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.encryptBuffer(256)
	}
}

func BenchmarkEncryptAndDecrypt(b *testing.B) {
	var err error

	s := &serverConnection{
		authenticator:  dummyAuthenticator{},
		rawEntryReader: dummyRawEntryReader{},
	}

	aes, err := aes.NewCipher(s.clientSecret[:])

	if err != nil {
		b.Fatal(err)
	}

	if s.encrypt, err = cipher.NewGCM(aes); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.encryptBuffer(256)
		s.decryptBuffer(272)
	}
}
