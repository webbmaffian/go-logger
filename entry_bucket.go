package logger

type Bucket uint32

func (b Bucket) writeEntry(e *Entry) {
	e.BucketId = uint32(b)
}
