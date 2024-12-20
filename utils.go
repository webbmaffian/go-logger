package logger

import (
	"regexp"
	"strconv"
	"strings"
)

var regexErrorString = regexp.MustCompile(`('[^']+')|([0-9]+\.?[0-9]*)`)

func parseErrorString(e *Entry, str string) {
	e.tagsCount = 0

	e.message = truncate(regexErrorString.ReplaceAllStringFunc(str, func(s string) string {
		if len(s) > 32 || e.tagsCount >= 8 {
			return s
		}

		e.tags[e.tagsCount] = strings.Trim(s, "'. ")
		e.tagsCount++

		return "%s"
	}), MaxMessageSize)
}

func max[T ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~int8 | ~int16 | ~int32 | ~int64 | ~int | ~uint | ~float32 | ~float64](a, b T) T {
	if a > b {
		return a
	}

	return b
}

func min[T ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~int8 | ~int16 | ~int32 | ~int64 | ~int | ~uint | ~float32 | ~float64](a, b T) T {
	if a < b {
		return a
	}

	return b
}

func truncate(str string, length int) string {
	if len(str) > length {
		return str[:length]
	}

	return str
}

type stringer interface {
	String() string
}

func stringify(val any) string {
	switch v := val.(type) {
	case string:
		return v
	case stringer:
		return v.String()
	case error:
		return v.Error()
	case bool:
		if v {
			return "true"
		} else {
			return "false"
		}
	case []byte:
		return b2s(v)
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', 6, 32)
	case float64:
		return strconv.FormatFloat(v, 'f', 6, 64)

	}

	return ""
}
