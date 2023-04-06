package logger

import (
	"github.com/rs/xid"
)

type Logger struct {
	tags         []string
	metaKeys     []string
	metaValues   []string
	metricKeys   []string
	metricValues []int32
	pool         *LoggerPool
	ttlEntry     uint16
	ttlMeta      uint16
	categoryId   uint8
}

func (l *Logger) Reset() {
	l.tags = l.tags[:0]
	l.metaKeys = l.metaKeys[:0]
	l.metaValues = l.metaValues[:0]
	l.metricKeys = l.metricKeys[:0]
	l.metricValues = l.metricValues[:0]
	l.ttlEntry = l.pool.DefaultEntryTTL
	l.ttlMeta = l.pool.DefaultMetaTTL
	l.categoryId = 0
}

func (l *Logger) Release() {
	l.pool.Release(l)
}

// System is unusable - a panic condition
func (l *Logger) Emerg(message string, tags ...string) (e *Entry) {
	return l.log(EMERG, message, tags)
}

// Action must be taken immediately, e.g. corrupted system database, or backup failures
func (l *Logger) Alert(message string, tags ...string) (e *Entry) {
	return l.log(ALERT, message, tags)
}

// Critical condition that prevents a specific task, e.g. fatal error
func (l *Logger) Crit(message string, tags ...string) (e *Entry) {
	return l.log(CRIT, message, tags)
}

// Non-critical errors, but that must be fixed
func (l *Logger) Err(message string, tags ...string) (e *Entry) {
	return l.log(ERR, message, tags)
}

// Warnings about unexpected conditions that might lead to errors further on
func (l *Logger) Warning(message string, tags ...string) (e *Entry) {
	return l.log(WARNING, message, tags)
}

// Normal but significant conditions
func (l *Logger) Notice(message string, tags ...string) (e *Entry) {
	return l.log(NOTICE, message, tags)
}

// Informational events of normal operations, e.g. taken actions or user errors
func (l *Logger) Info(message string, tags ...string) (e *Entry) {
	return l.log(INFO, message, tags)
}

// Helpful information for troubleshooting
func (l *Logger) Debug(message string, tags ...string) (e *Entry) {
	return l.log(DEBUG, message, tags)
}

func (l *Logger) log(severity Severity, message string, tags []string) (e *Entry) {
	e = l.pool.EntryPool.Acquire()
	e.logger = l
	e.id = xid.NewWithTime(l.pool.Clock.Now())
	e.severity = severity
	e.message = truncate(message, MaxMessageSize)
	e.ttlEntry = l.ttlEntry
	e.ttlMeta = l.ttlMeta
	e.categoryId = l.categoryId
	copy(e.tags[:], tags)
	return
}

func (l *Logger) Send(err error) xid.ID {
	return l.pool.Send(err)
}

func (l *Logger) Category(categoryId uint8) *Logger {
	l.categoryId = categoryId
	return l
}

func (l *Logger) Tag(tags ...string) *Logger {
	l.tags = append(l.tags, tags...)

	return l
}

func (l *Logger) Meta(key string, value string) *Logger {
	l.metaKeys = append(l.metaKeys, truncate(key, MaxMetaKeySize))
	l.metaValues = append(l.metaValues, truncate(key, MaxMetaKeySize))

	return l
}

func (l *Logger) Metric(key string, value int32) *Logger {
	l.metricKeys = append(l.metricKeys, truncate(key, MaxMetaKeySize))
	l.metricValues = append(l.metricValues, value)

	return l
}

func (l *Logger) TTL(ttl int) *Logger {
	l.ttlEntry = uint16(ttl)
	return l
}

func (l *Logger) MetaTTL(ttl int) *Logger {
	l.ttlMeta = uint16(ttl)
	return l
}

func (l *Logger) Logger() (l2 *Logger) {
	l2 = l.pool.Logger()
	l2.categoryId = l.categoryId
	l2.metaKeys = append(l2.metaKeys, l.metaKeys...)
	l2.metaValues = append(l2.metaValues, l.metaValues...)
	l2.metricKeys = append(l2.metricKeys, l.metricKeys...)
	l2.metricValues = append(l2.metricValues, l.metricValues...)
	l2.tags = append(l2.tags, l.tags...)
	l2.ttlEntry = l.ttlEntry
	l2.ttlMeta = l.ttlMeta
	return l2
}
