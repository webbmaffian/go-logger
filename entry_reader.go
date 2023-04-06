package logger

import (
	"time"

	"github.com/rs/xid"
)

type entryReader struct {
	e *Entry
}

func (r entryReader) Bucket() uint32 {
	return r.e.bucketId
}

func (r entryReader) Id() xid.ID {
	return r.e.id
}

func (r entryReader) Msg() string {
	return r.e.message
}

func (r entryReader) Sev() Severity {
	return r.e.severity
}

func (r entryReader) Time() time.Time {
	return r.e.id.Time()
}

func (r entryReader) Tags() []string {
	return r.e.tags[:r.e.tagsCount]
}

func (r entryReader) Cat() uint8 {
	return r.e.categoryId
}

func (r entryReader) Meta() (keys []string, values []string) {
	return r.e.metaKeys[:r.e.metaCount], r.e.metaValues[:r.e.metaCount]
}

func (r entryReader) Metrics() (keys []string, values []int32) {
	return r.e.metricKeys[:r.e.metricCount], r.e.metricValues[:r.e.metricCount]
}

func (r entryReader) Trace() (paths []string, lines []uint16) {
	return r.e.stackTracePaths[:r.e.stackTraceCount], r.e.stackTraceLines[:r.e.stackTraceCount]
}

func (r entryReader) TTL() uint16 {
	return r.e.ttlEntry
}

func (r entryReader) MetaTTL() uint16 {
	return r.e.ttlMeta
}

func (r entryReader) HasId() bool {
	return !r.e.id.IsNil()
}

func (r entryReader) HasTags() bool {
	return r.e.tagsCount != 0
}

func (r entryReader) HasCat() bool {
	return r.e.categoryId != 0
}

func (r entryReader) HasMeta() bool {
	return r.e.metaCount != 0
}

func (r entryReader) HasMetrics() bool {
	return r.e.metricCount != 0
}

func (r entryReader) HasTrace() bool {
	return r.e.stackTraceCount != 0
}

func (r entryReader) FullTags() bool {
	return r.e.tagsCount == MaxTagsCount
}

func (r entryReader) FullMeta() bool {
	return r.e.metaCount == MaxMetaCount
}

func (r entryReader) FullMetrics() bool {
	return r.e.metricCount != MaxMetricCount
}
