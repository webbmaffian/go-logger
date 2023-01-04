package logger

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
)

type serverConnection struct {
	buf            [entrySize]byte
	clientSecret   Secret
	clientId       uuid.UUID
	sizeBuf        [2]byte
	tenantId       uint32
	encrypt        cipher.AEAD
	authenticator  Authenticator
	rawEntryReader RawEntryReader
}

func (s *serverConnection) decryptBuffer(encryptedSize int) (rawSize int, err error) {
	_, err = s.encrypt.Open(s.buf[12:12], s.buf[:12], s.buf[12:encryptedSize], nil)
	rawSize = encryptedSize - s.encrypt.Overhead()

	return
}

func (s *serverConnection) encryptBuffer(rawSize int) (encryptedSize int) {
	s.encrypt.Seal(s.buf[12:12], s.buf[:12], s.buf[12:rawSize], nil)

	return rawSize + s.encrypt.Overhead()
}

func (s *serverConnection) authenticate(ctx context.Context, conn net.Conn) (err error) {

	// Wait for data - abort after 5 seconds
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetReadDeadline(time.Time{})

	// Wait for client ID
	if _, err = io.ReadFull(conn, s.clientId[:]); err != nil {
		return
	}

	if s.authenticator != nil {
		if s.tenantId, s.clientSecret, err = s.authenticator.LoadClientSecret(ctx, s.clientId); err != nil {
			return
		}
	}

	aes, err := aes.NewCipher(s.clientSecret[:])

	if err != nil {
		return
	}

	s.encrypt, err = cipher.NewGCM(aes)

	if err != nil {
		return
	}

	// Generate nonce
	if _, err = rand.Read(s.buf[:12]); err != nil {
		return
	}

	// Wait for challenge
	if _, err = io.ReadFull(conn, s.buf[12:28]); err != nil {
		return
	}

	size := s.encryptBuffer(28)
	_, err = conn.Write(s.buf[:size])

	return
}

func (s *serverConnection) readEntries(ctx context.Context, r io.Reader) (err error) {
	if err = ctx.Err(); err != nil {
		return
	}

	if _, err = readFull(ctx, r, s.sizeBuf[:]); err != nil {
		return
	}

	size := int(binary.BigEndian.Uint16(s.sizeBuf[:]))

	if size < 28 {
		return fmt.Errorf("too short message: only %d bytes, expected at least 28 bytes", size)
	}

	if _, err = readFull(ctx, r, s.buf[:size]); err != nil {
		return
	}

	if size, err = s.decryptBuffer(size); err != nil {
		return
	}

	if err = validateEntryBytes(s.buf[:size]); err != nil {
		return
	}

	if s.rawEntryReader != nil {
		if err = s.rawEntryReader.Read(s.tenantId, s.buf[:size]); err != nil {
			log.Println("Invalid entry:", err)
			return
		}
	}

	return
}
