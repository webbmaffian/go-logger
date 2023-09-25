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
)

var errPonged = errors.New("ponged")

type tlsServerConn struct {
	buf              [logger.MaxEntrySize]byte
	validBucketIds   []byte
	entryProc        logger.EntryProcessor
	clock            fastime.Fastime
	entry            *logger.Entry
	conn             *tls.Conn
	log              *logger.Logger
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
			if conn.ack {
				if netErr, ok := err.(net.Error); !ok || !netErr.Timeout() {
					if err := conn.sendAck(respAckNOK); err != nil {
						conn.log.Send(err)
					}
				}
			}

			return
		}

		if conn.ack {
			if err := conn.sendAck(respAckOK); err != nil {
				conn.log.Send(err)
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
		conn.pingsReceived++
		_, err = conn.conn.Write([]byte{1})

		if err == nil {
			conn.pongsSent++
			return errPonged
		}
	}

	conn.entriesReceived++

	if size < 6 {
		return logger.ErrTooShort
	}

	if _, err = io.ReadFull(conn.conn, conn.buf[2:size]); err != nil {
		return
	}

	if !conn.validBucketId() {
		return logger.ErrForbiddenBucket
	}

	if err = conn.entry.Decode(conn.buf[:size], conn.noCopy); err != nil {
		return
	}

	if err = conn.entryProc.ProcessEntry(ctx, conn.entry); err != nil {
		err = conn.log.Err("Failed to process entry %s", conn.entry.Read().Id()).MetaBlob(err.Error())
	} else {
		conn.entriesSucceeded++
	}

	return
}

func (conn *tlsServerConn) sendAck(ack respType) (err error) {
	_, err = conn.conn.Write([]byte{byte(ack)})

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
