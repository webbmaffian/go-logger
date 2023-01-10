package logger

import (
	"context"
	"encoding/binary"
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
					s := e.Encode(buf[2:])
					binary.BigEndian.PutUint16(buf[:], uint16(s))

					if _, err := output.Write(buf[:s+2]); err != nil {
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

	e.id = xid.NewWithTime(l.opt.TimeNow())
	e.level = 3
	e.severity = severity
	e.message = truncate(message, math.MaxUint8)
	e.tagsCount = 0
	e.metaCount = 0

	for i := range args {
		switch v := args[i].(type) {

		case Category:
			e.category = truncate(string(v), math.MaxUint8)
			e.level = max(e.level, 3)

		case ProcId:
			e.procId = truncate(string(v), math.MaxUint8)
			e.level = max(e.level, 4)

		case string:
			if e.tagsCount < 32 {
				e.tags[e.tagsCount] = truncate(v, math.MaxUint8)
				e.tagsCount++
				e.level = max(e.level, 5)
			}

		case int:
			if e.tagsCount < 32 {
				e.tags[e.tagsCount] = strconv.Itoa(v)
				e.tagsCount++
				e.level = max(e.level, 5)
			}

		case meta:
			if e.metaCount < 32 {
				e.metaKeys[e.metaCount] = truncate(v.key, math.MaxUint8)
				e.metaValues[e.metaCount] = truncate(v.value, math.MaxUint16)
				e.metaCount++
				e.level = max(e.level, 6)
			}

		}
	}

	l.queue.putEntry(e)

	return e.id
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
