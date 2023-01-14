package logger

import (
	"context"
	"io"
	"reflect"
	"regexp"
	"strings"
	"unsafe"
)

var regexErrorString = regexp.MustCompile("('[^']+')|([0-9]+\\.?[0-9]*)")

func parseErrorString(e *Entry, str string) {
	e.TagsCount = 0

	regexErrorString.ReplaceAllStringFunc(str, func(s string) string {
		if len(s) > 32 || e.TagsCount >= 8 {
			return s
		}

		e.Tags[e.TagsCount] = strings.Trim(s, "'. ")
		e.TagsCount++

		return "%s"
	})
}

func max[T uint8 | uint16 | uint32 | uint64 | int8 | int16 | int32 | int64 | int | uint | float32 | float64 | level](a, b T) T {
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

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func stringToBytes(s string) (b []byte) {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Cap = sh.Len
	bh.Len = sh.Len
	return b
}
