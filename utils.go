package logger

import (
	"context"
	"errors"
	"io"
	"log"
	"math/big"
	"net"
	"regexp"
	"strconv"
	"strings"
	"unsafe"

	"github.com/google/uuid"
)

var regexErrorString = regexp.MustCompile(`('[^']+')|([0-9]+\.?[0-9]*)`)

func parseErrorString(e *Entry, str string) {
	e.tagsCount = 0

	e.message = truncate(regexErrorString.ReplaceAllStringFunc(str, func(s string) string {
		if len(s) > 32 || e.tagsCount >= 8 {
			return s
		}

		e.tags[e.tagsCount] = strings.Trim(s, "'. ")
		e.tagsCount++

		return "%s"
	}), MaxMessageSize)
}

func max[T uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64 | int | uint | float32 | float64 | level](a, b T) T {
	if a > b {
		return a
	}

	return b
}

func min[T uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64 | int | uint | float32 | float64 | level](a, b T) T {
	if a < b {
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
		log.Println("Read", n, "bytes")
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

func stringToBytes(s string) (b []byte) {
	return *(*[]byte)(unsafe.Pointer(&struct {
		string
		int
	}{s, len(s)}))
}

type stringer interface {
	String() string
}

func stringify(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case stringer:
		return v.String()
	case error:
		return v.Error()
	case bool:
		if v {
			return "true"
		} else {
			return "false"
		}
	case []byte:
		return bytesToString(v)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', 6, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', 6, 64)

	}

	return ""
}

var ErrNaN = errors.New("not a number")

func toInt32(val any) (n int32, err error) {
	switch v := val.(type) {
	case int:
		n = int32(v)
	case int8:
		n = int32(v)
	case int16:
		n = int32(v)
	case int32:
		n = int32(v)
	case int64:
		n = int32(v)
	case uint:
		n = int32(v)
	case uint8:
		n = int32(v)
	case uint16:
		n = int32(v)
	case uint32:
		n = int32(v)
	case uint64:
		n = int32(v)
	case float32:
		n = int32(v)
	case float64:
		n = int32(v)
	default:
		err = ErrNaN
	}

	return
}

func addrToIp(addr net.Addr) net.IP {
	switch addr := addr.(type) {
	case *net.UDPAddr:
		return addr.IP
	case *net.TCPAddr:
		return addr.IP
	}

	return nil
}

func bigIntToUuid(i *big.Int) (id uuid.UUID) {
	i.FillBytes(id[:])
	return
}
