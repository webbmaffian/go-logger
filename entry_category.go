package logger

type Category uint8

func (c Category) writeEntry(e *Entry) {
	e.CategoryId = uint8(c)
}
