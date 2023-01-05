package remote

import (
	"context"
	"io"
)

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
