package logger

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Client interface {
	Write(e *entry)
	Close() error
	AcquireEntry() *entry
}

type ClientOptions struct {
	Host         string
	Port         int
	ClientId     []byte
	ClientSecret []byte
	Buffer       int
}

type client struct {
	opt          ClientOptions
	ctx          context.Context
	wg           sync.WaitGroup
	conn         net.Conn
	ch           chan *entry
	entryPool    sync.Pool
	buf          [entrySize]byte
	clientId     [16]byte
	clientSecret [32]byte
	encrypt      cipher.AEAD
	connected    bool
	closed       bool
}

func NewClient(ctx context.Context, opt ClientOptions) (Client, error) {
	var err error

	c := &client{
		ctx: ctx,
		opt: opt,
		ch:  make(chan *entry, opt.Buffer),
		entryPool: sync.Pool{
			New: func() any {
				return new(entry)
			},
		},
	}

	aes, err := aes.NewCipher(c.clientSecret[:])

	if err != nil {
		return nil, err
	}

	c.encrypt, err = cipher.NewGCM(aes)

	if err != nil {
		return nil, err
	}

	c.wg.Add(1)
	go c.worker()

	return c, err
}

func (c *client) AcquireEntry() *entry {
	return c.entryPool.Get().(*entry)
}

func (c *client) Write(e *entry) {
	if c.closed {
		return
	}

	// Todo: Check size of channel and handle fallback
	c.ch <- e
}

func (c *client) worker() {
	for e := range c.ch {
		c.workEntry(e)
	}

	c.wg.Done()
}

func (c *client) workEntry(e *entry) {
	var err error

	c.ensureConnection()

	size := e.encode(c.buf[2:])
	size += c.encryptBuffer(c.buf[2:14], c.buf[14:size+2])
	c.buf[0] = byte(size >> 8)
	c.buf[1] = byte(size)

	_, err = c.conn.Write(c.buf[:size+2])

	if err != nil {
		c.ensureConnection()
		_, err = c.conn.Write(c.buf[:size+2])
	}

	c.entryPool.Put(e)
}

func (c *client) encryptBuffer(nonce []byte, buf []byte) (encryptedSize int) {
	c.encrypt.Seal(buf[:0], nonce, buf, nil)

	return c.encrypt.Overhead()
}

func (c *client) Close() (err error) {
	log.Println("Closing client")
	c.closed = true
	close(c.ch)
	c.wg.Wait()
	c.connected = false
	return c.conn.Close()
}

func (c *client) ensureConnection() {
	if c.connected {
		return
	}

	var ticker *time.Ticker

loop:
	for {
		if err := c.connect(); err == nil {
			break loop
		}

		if ticker == nil {
			ticker = time.NewTicker(time.Second * 5)
		}

		select {
		case <-c.ctx.Done():
			break loop
		case <-ticker.C:
			continue
		}
	}

	if ticker != nil {
		ticker.Stop()
	}
}

func (c *client) connect() (err error) {
	var address strings.Builder
	address.Grow(len(c.opt.Host) + 5)
	address.WriteString(c.opt.Host)
	address.WriteByte(':')
	address.WriteString(strconv.Itoa(c.opt.Port))

	if c.conn != nil {
		c.conn.Close()
	}

	c.connected = false

	ctx, cancel := context.WithTimeout(c.ctx, time.Second*5)
	defer cancel()

	var d net.Dialer

	if c.conn, err = d.DialContext(ctx, "tcp", address.String()); err != nil {
		return
	}

	if err = c.authenticate(); err != nil {
		c.conn.Close()
	} else {
		c.connected = true
	}

	return
}

func (c *client) authenticate() (err error) {
	var challenge [16]byte
	if _, err = rand.Read(challenge[:]); err != nil {
		return
	}

	// Write client ID
	c.conn.Write(c.clientId[:])

	// Write challenge
	c.conn.Write(challenge[:])

	// Wait for response - abort after 5 seconds
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer c.conn.SetReadDeadline(time.Time{})

	var resp [44]byte // 12 byte nonce + 16 byte challange + 16 byte overhead
	if _, err = io.ReadFull(c.conn, resp[:]); err != nil {
		return
	}

	if _, err = c.encrypt.Open(resp[12:12], resp[:12], resp[12:], nil); err != nil {
		return
	}

	if !bytes.Equal(challenge[:], resp[12:28]) {
		return errors.New("Invalid handshake from server")
	}

	return
}
