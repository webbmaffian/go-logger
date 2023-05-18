package peer

import (
	"context"
	"log"

	"github.com/webbmaffian/go-logger"
)

var _ logger.EntryProcessor = entryEchoer{}

type entryEchoer struct{}

func (entryEchoer) ProcessEntry(_ context.Context, e *logger.Entry) error {
	log.Println(e.Read().Time(), "-", e.String())
	return nil
}
