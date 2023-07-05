package peer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"github.com/kpango/fastime"
	"github.com/webbmaffian/go-logger"
)

var ponged = errors.New("ponged")

type tlsServerConn struct {
	buf            [logger.MaxEntrySize]byte
	validBucketIds []byte
	entryProc      logger.EntryProcessor
	clock          fastime.Fastime
	entry          *logger.Entry
	conn           *tls.Conn
	clientTimeout  time.Duration
	noCopy         bool
	ack            bool
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

	size := binary.BigEndian.Uint16(conn.buf[:2])

	// Sending two empty bytes is a ping - answer with a 1 byte pong
	if size == 0 {
		_, err = conn.conn.Write([]byte{1})

		if err == nil {
			return ponged
		}
	}

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

	return conn.entryProc.ProcessEntry(ctx, conn.entry)
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
