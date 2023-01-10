package logger

import (
	"context"
	"io"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/rs/xid"
)

type LoggerOptions struct {
	TimeNow func() time.Time
}

func New(ctx context.Context, output io.WriteCloser, options ...LoggerOptions) Logger {
	var opt LoggerOptions

	if options != nil {
		opt = options[0]
	}

	if opt.TimeNow == nil {
		opt.TimeNow = time.Now
	}

	queue := newEntryQueue(100)

	go func() {
		var buf [entrySize]byte

	loop:
		for {
			select {
			case err := <-ctx.Done():
				log.Println(err)
				break loop
			case e, ok := <-queue.ch:
				if ok {
					s := e.Encode(buf[:])

					if _, err := output.Write(buf[:s]); err != nil {
						log.Println(err)
					}
				}

				queue.releaseEntry(e)
			}
		}

		output.Close()
	}()

	return Logger{
		ctx:   ctx,
		queue: queue,
		opt:   opt,
	}
}

type Logger struct {
	ctx   context.Context
	queue *entryQueue
	opt   LoggerOptions
}

func (l *Logger) Emerg(message string, args ...any) xid.ID {
	return l.log(EMERG, message, args...)
}

func (l *Logger) Alert(message string, args ...any) xid.ID {
	return l.log(ALERT, message, args...)
}

func (l *Logger) Crit(message string, args ...any) xid.ID {
	return l.log(CRIT, message, args...)
}

func (l *Logger) Err(message string, args ...any) xid.ID {
	return l.log(ERR, message, args...)
}

func (l *Logger) Warning(message string, args ...any) xid.ID {
	return l.log(WARNING, message, args...)
}

func (l *Logger) Notice(message string, args ...any) xid.ID {
	return l.log(NOTICE, message, args...)
}

func (l *Logger) Info(message string, args ...any) xid.ID {
	return l.log(INFO, message, args...)
}

func (l *Logger) Debug(message string, args ...any) xid.ID {
	return l.log(DEBUG, message, args...)
}

func (l *Logger) log(severity Severity, message string, args ...any) xid.ID {
	e := l.queue.acquireEntry()

	e.Id = xid.NewWithTime(l.opt.TimeNow())
	e.Level = 3
	e.Severity = severity
	e.Message = truncate(message, math.MaxUint8)
	e.TagsCount = 0
	e.MetaCount = 0

	for i := range args {
		switch v := args[i].(type) {

		case Category:
			e.Category = truncate(string(v), math.MaxUint8)
			e.Level = max(e.Level, 3)

		case ProcId:
			e.ProcId = truncate(string(v), math.MaxUint8)
			e.Level = max(e.Level, 4)

		case string:
			if e.TagsCount < 32 {
				e.Tags[e.TagsCount] = truncate(v, math.MaxUint8)
				e.TagsCount++
				e.Level = max(e.Level, 5)
			}

		case int:
			if e.TagsCount < 32 {
				e.Tags[e.TagsCount] = strconv.Itoa(v)
				e.TagsCount++
				e.Level = max(e.Level, 5)
			}

		case meta:
			if e.MetaCount < 32 {
				e.MetaKeys[e.MetaCount] = truncate(v.key, math.MaxUint8)
				e.MetaValues[e.MetaCount] = truncate(v.value, math.MaxUint16)
				e.MetaCount++
				e.Level = max(e.Level, 6)
			}

		}
	}

	l.queue.putEntry(e)

	return e.Id
}

func (l *Logger) Close() {
	l.queue.close()
}

func Meta(key string, value string) meta {
	return meta{key, value}
}

type meta struct {
	key   string
	value string
}

type Category string

type ProcId string
