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
	pool         *Pool
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
	l.ttlEntry = l.pool.opt.DefaultEntryTTL
	l.ttlMeta = l.pool.opt.DefaultMetaTTL
	l.categoryId = 0
}

func (l *Logger) Drop() {
	l.pool.ReleaseLogger(l)
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
	e = l.pool.Entry()
	e.bucketId = l.pool.opt.BucketId
	e.logger = l
	e.id = xid.NewWithTime(l.pool.client.Now())
	e.severity = severity
	e.message = truncate(message, MaxMessageSize)
	e.ttlEntry = l.ttlEntry
	e.ttlMeta = l.ttlMeta
	e.categoryId = l.categoryId
	copy(e.tags[:], tags)
	return
}

// Send an error to the log
func (l *Logger) Send(err error) xid.ID {
	return l.pool.Send(err)
}

// Set the default category ID for this logger. All entries created from this logger will have
// this category ID if not overridden.
func (l *Logger) Cat(categoryId uint8) *Logger {
	l.categoryId = categoryId
	return l
}

// Set tags for this logger. All entries created from this logger will have these tags appended.
func (l *Logger) Tag(tags ...string) *Logger {
	l.tags = append(l.tags, tags...)

	return l
}

// Set meta data for this logger. All entries created from this logger will have these meta data appended.
func (l *Logger) Meta(key string, value string) *Logger {
	l.metaKeys = append(l.metaKeys, truncate(key, MaxMetaKeySize))
	l.metaValues = append(l.metaValues, truncate(key, MaxMetaKeySize))

	return l
}

// Set metrics for this logger. All entries created from this logger will have these metrics appended.
func (l *Logger) Metric(key string, value int32) *Logger {
	l.metricKeys = append(l.metricKeys, truncate(key, MaxMetaKeySize))
	l.metricValues = append(l.metricValues, value)

	return l
}

// Set the default TTL for this logger. All entries created from this logger will have
// this TTL if not overridden.
func (l *Logger) TTL(days int) *Logger {
	l.ttlEntry = uint16(days)
	return l
}

// Set the default meta TTL for this logger. All entries created from this logger will have
// this meta TTL if not overridden.
func (l *Logger) MetaTTL(days int) *Logger {
	l.ttlMeta = uint16(days)
	return l
}

// Create a new logger taht will inherit any context from this logger.
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
