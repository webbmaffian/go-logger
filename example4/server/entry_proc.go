package main

import (
	"context"
	"fmt"
	"os"

	"github.com/webbmaffian/go-logger"
)

type entryEchoer struct {
	count int
}

func (ee *entryEchoer) ProcessEntry(_ context.Context, e *logger.Entry) error {
	ee.count++
	os.Stdout.WriteString(fmt.Sprintf("Received %4d\r", ee.count))
	return nil
}
