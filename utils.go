package logger

import (
	"regexp"
	"strconv"
	"strings"
	"unsafe"
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

func truncate(str string, length int) string {
	if len(str) > length {
		return str[:length]
	}

	return str
}

func bytesToString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func stringToBytes(s string) (b []byte) {
	return *(*[]byte)(unsafe.Pointer(&struct {
		string
		int
	}{s, len(s)}))
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
		return bytesToString(v)
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
