package main

import (
	"context"
	"log"

	"github.com/webbmaffian/go-logger"
	"github.com/webbmaffian/go-logger/example3"
)

type entryEchoer struct{}

func (entryEchoer) ProcessEntry(_ context.Context, e *logger.Entry) error {
	log.Println("\n" + example3.FormatEntry(e, ">"))
	return nil
}
