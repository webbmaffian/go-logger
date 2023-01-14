package logger

type Category string

func (c Category) writeEntry(e *Entry) {
	e.Category = string(c)
}
