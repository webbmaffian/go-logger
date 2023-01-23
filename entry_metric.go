package logger

import "sync"

var metricPool = sync.Pool{
	New: func() any {
		return new(metric)
	},
}

// Key-value pairs of metrics
func Metric(key string, value int32) (m metric) {
	// m = metricPool.Get().(*metric)
	m.key = key
	m.value = value
	return
}

type metric struct {
	key   string
	value int32
}

func (m metric) writeEntry(e *Entry) {
	e.Level = max(e.Level, _6_Metric)

	if e.MetricCount < MaxMetricCount {
		e.MetricKeys[e.MetricCount] = truncate(m.key, MaxMetaKeySize)
		e.MetricValues[e.MetricCount] = m.value
		e.MetricCount++
	}

	// metricPool.Put(m)
}
