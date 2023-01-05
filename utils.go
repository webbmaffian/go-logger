package logger

import (
	"context"
	"io"
	"net"
	"unsafe"
)

func max[T uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64 | int | uint | float32 | float64](a, b T) T {
	if a > b {
		return a
	}

	return b
}

func truncate(str string, length int) string {
	if len(str) > length {
		return str[:length]
	}

	return str
}

func readFull(ctx context.Context, r io.Reader, buf []byte) (n int, err error) {
	min := len(buf)
	for n < min && err == nil {
		if err = ctx.Err(); err != nil {
			return
		}
		var nn int
		nn, err = r.Read(buf[n:])
		n += nn
	}
	if n >= min {
		err = nil
	} else if err == io.EOF {
		if n > 0 {
			err = io.ErrUnexpectedEOF
		} else {
			err = nil
		}
	}
	return
}

func readFullPackets(ctx context.Context, r net.PacketConn, buf []byte) (n int, err error) {
	min := len(buf)
	for n < min && err == nil {
		if err = ctx.Err(); err != nil {
			return
		}
		var nn int
		nn, _, err = r.ReadFrom(buf[n:])
		n += nn
	}
	if n >= min {
		err = nil
	} else if err == io.EOF {
		if n > 0 {
			err = io.ErrUnexpectedEOF
		} else {
			err = nil
		}
	}
	return
}

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func stringToBytes(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}
