package example3

import (
	"fmt"
	"io"
	"strings"

	"github.com/webbmaffian/go-logger"
)

func FormatEntry(e *logger.Entry, rowPrefix string) string {
	var b strings.Builder

	f := entryFormatter{
		w:         &b,
		keyPad:    -16,
		rowPrefix: rowPrefix,
	}

	r := e.Read()

	f.write("ID", r.Id())
	f.write("STRING", e.String())
	f.write("MSG", r.Msg())
	f.write("TAGS", r.Tags())

	keys, values := r.Meta()
	f.write("META KEYS", keys)
	f.write("META VALUES", values)

	return b.String()
}

type entryFormatter struct {
	rowPrefix string
	keyPad    int
	w         io.Writer
}

func (f entryFormatter) write(key string, value any) {
	fmt.Fprintf(f.w, "%s %*s| %v |\n", f.rowPrefix, f.keyPad, key, value)
}
