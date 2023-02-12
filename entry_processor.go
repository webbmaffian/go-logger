package logger

type EntryProcessor interface {
	ProcessEntry(entry *Entry) (err error)
}
