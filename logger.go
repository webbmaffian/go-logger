package logger

import (
	"context"
	"io"
	"log"
	"math"
	"time"

	"github.com/rs/xid"
)

type LoggerOptions struct {
	TimeNow            func() time.Time
	EntryQueueSize     int
	BucketId           uint32
	DefaultEntryTTL    uint16
	DefaultMetaTTL     uint16
	StackTraceSeverity Severity
}

func New(ctx context.Context, output io.WriteCloser, options ...LoggerOptions) Logger {
	var opt LoggerOptions

	if options != nil {
		opt = options[0]
	}

	if opt.TimeNow == nil {
		opt.TimeNow = time.Now
	}

	if opt.EntryQueueSize <= 0 {
		opt.EntryQueueSize = 100
	}

	queue := newEntryQueue(opt.EntryQueueSize)

	go func() {
		var buf [MaxEntrySize]byte

	loop:
		for {
			select {
			case err := <-ctx.Done():
				log.Println(err)
				break loop
			case e, ok := <-queue.ch:
				if ok {
					s := e.Encode(buf[:])
					_ = s

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

// System is unusable - a panic condition
func (l *Logger) Emerg(message string, args ...any) {
	l.log(EMERG, message, args)
}

// Action must be taken immediately, e.g. corrupted system database, or backup failures
func (l *Logger) Alert(message string, args ...any) {
	l.log(ALERT, message, args)
}

// Critical condition that prevents a specific task, e.g. fatal error
func (l *Logger) Crit(message string, args ...any) {
	l.log(CRIT, message, args)
}

// Non-critical errors, but that must be fixed
func (l *Logger) Err(message string, args ...any) {
	l.log(ERR, message, args)
}

// Warnings about unexpected conditions that might lead to errors further on
func (l *Logger) Warning(message string, args ...any) {
	l.log(WARNING, message, args)
}

// Normal but significant conditions
func (l *Logger) Notice(message string, args ...any) {
	l.log(NOTICE, message, args)
}

// Informational events of normal operations, e.g. taken actions or user errors
func (l *Logger) Info(message string, args ...any) {
	l.log(INFO, message, args)
}

// Helpful information for troubleshooting
func (l *Logger) Debug(message string, args ...any) {
	l.log(DEBUG, message, args)
}

func (l *Logger) LogError(err error) (entryId xid.ID) {
	err = l.NewError(err)

	if e, ok := err.(*Entry); ok {
		l.queue.putEntry(e)
		entryId = e.Id
	}

	return
}

func (l *Logger) NewError(err any, args ...any) error {
	if err == nil {
		return nil
	}

	var e *Entry

	switch v := err.(type) {

	case *Entry:
		e = v

	case Severitier:
		e = l.acquireEntry(v.Severity())
		parseErrorString(e, v.Error())

	case error:
		e = l.acquireEntry(ERR)
		parseErrorString(e, v.Error())

	case string:
		e = l.acquireEntry(ERR)
	}

	e.parseArgs(args)

	if e.Severity <= l.opt.StackTraceSeverity && e.StackTraceCount == 0 {
		e.addStackTrace(3)
	}

	return e
}

func (l *Logger) log(severity Severity, message string, args []any) {
	e := l.newEntry(severity, message, args)

	if severity <= l.opt.StackTraceSeverity {
		e.addStackTrace(4)
	}

	l.queue.putEntry(e)
	// l.queue.releaseEntry(e)
}

func (l *Logger) newEntry(severity Severity, message string, args []any) *Entry {
	e := l.acquireEntry(severity)
	e.Message = truncate(message, math.MaxUint8)
	e.parseArgs(args)

	return e
}

func (l *Logger) acquireEntry(sev Severity) *Entry {
	e := l.queue.acquireEntry(l.opt.TimeNow())
	e.BucketId = l.opt.BucketId
	e.Severity = sev
	e.TtlEntry = l.opt.DefaultEntryTTL
	e.TtlMeta = l.opt.DefaultMetaTTL

	return e
}

func (l *Logger) Close() {
	l.queue.close()
}
