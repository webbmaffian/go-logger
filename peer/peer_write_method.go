package peer

type WriteMethod uint8

const (
	WriteOrReplace WriteMethod = iota // The oldest log entry in the buffer will be replaced with the new.
	WriteOrBlock                      // The log method will block until there is available space in the buffer.
	WriteOrFail                       // The log method won't do anything if the buffer is full.
)
