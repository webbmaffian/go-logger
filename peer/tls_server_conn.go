package peer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"time"

	"github.com/kpango/fastime"
	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/debug"
)

var ponged = errors.New("ponged")

type tlsServerConn struct {
	buf              [logger.MaxEntrySize]byte
	validBucketIds   []byte
	entryProc        logger.EntryProcessor
	clock            fastime.Fastime
	entry            *logger.Entry
	conn             *tls.Conn
	log              *logger.Logger
	debugSocket      *debug.Writer
	clientTimeout    time.Duration
	timeConnected    int64
	timeLastActive   int64
	pingsReceived    int32
	pongsSent        int32
	entriesReceived  int32
	entriesSucceeded int32
	noCopy           bool
	ack              bool
}

func (conn *tlsServerConn) listen(ctx context.Context) (err error) {
	for {
		if err = conn.handleEntry(ctx); err != nil {
			return
		}

		if conn.ack {
			if err = conn.sendAck(); err != nil {
				return
			}
		}
	}
}

func (conn *tlsServerConn) handleEntry(ctx context.Context) (err error) {
	conn.conn.SetReadDeadline(conn.clock.Now().Add(conn.clientTimeout))

	if _, err = io.ReadFull(conn.conn, conn.buf[:2]); err != nil {
		return
	}

	conn.timeLastActive = conn.clock.UnixNow()
	size := binary.BigEndian.Uint16(conn.buf[:2])

	// Sending two empty bytes is a ping - answer with a 1 byte pong
	if size == 0 {
		conn.debug("%15s -> ping", conn.ip())
		conn.pingsReceived++
		_, err = conn.conn.Write([]byte{1})

		if err == nil {
			conn.pongsSent++
			return ponged
		}
	}

	conn.entriesReceived++

	if size < 6 {
		conn.debug("%15s -> msg size %d dropped (%s)", conn.ip(), size, logger.ErrTooShort)
		return logger.ErrTooShort
	}

	if _, err = io.ReadFull(conn.conn, conn.buf[2:size]); err != nil {
		conn.debug("%15s -> msg size %d failed (%s)", conn.ip(), size, err)
		return
	}

	conn.debug("%15s -> msg size %d: %v", conn.ip(), size, conn.buf[2:size])
	conn.debug("%15s -> msg size %d: %s", conn.ip(), size, conn.buf[2:size])

	if !conn.validBucketId() {
		conn.debug("%15s -> msg size %d forbidden (%s)", conn.ip(), size, logger.ErrForbiddenBucket)
		return logger.ErrForbiddenBucket
	}

	if err = conn.entry.Decode(conn.buf[:size], conn.noCopy); err != nil {
		conn.debug("%15s -> msg size %d decoding failed (%s)", conn.ip(), size, err)
		return
	}

	if err = conn.entryProc.ProcessEntry(ctx, conn.entry); err != nil {
		conn.debug("%15s -> msg size %d processing failed (%s)", conn.ip(), size, err)
		err = conn.log.Err("Failed to process entry %s", conn.entry.Read().Id()).MetaBlob(err.Error())
	} else {
		conn.entriesSucceeded++
	}

	return
}

func (conn *tlsServerConn) sendAck() (err error) {
	entryId := conn.buf[6 : 6+xidLen]
	written := 0

	for written < xidLen {
		s, err := conn.conn.Write(entryId[written:])
		written += s

		if err == nil {
			continue
		}

		if err != io.ErrShortWrite {
			return err
		}
	}

	return
}

func (conn *tlsServerConn) validBucketId() bool {
	if conn.validBucketIds == nil {
		return true
	}

	for i := 0; i < len(conn.validBucketIds); i += 4 {
		if bytes.Equal(conn.buf[2:6], conn.validBucketIds[i:i+4]) {
			return true
		}
	}

	return false
}

func (conn *tlsServerConn) debug(format string, args ...any) {
	if conn.debugSocket != nil {
		conn.debugSocket.Write(format+"\n", args...)
	}
}

func (conn *tlsServerConn) ip() net.IP {
	return addrIp(conn.conn.RemoteAddr())
}
