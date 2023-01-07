package logger

import (
	"context"
	"time"

	"github.com/kpango/fastime"
)

func FastTimeNow(ctx context.Context) func() time.Time {
	t := fastime.New().StartTimerD(ctx, time.Second)
	return t.Now
}
