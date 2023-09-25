package channel

import (
	"io"
	"sync"
)

type ByteChannel struct {
	data          []byte
	readCond      sync.Cond // Awaited by readers, notified by writers.
	writeCond     sync.Cond // Awaited by writers, notified by readers.
	mu            sync.Mutex
	itemSize      int64
	startIdx      int64
	awaitingAck   int64
	length        int64
	capacity      int64
	itemsWritten  uint64
	itemsRead     uint64
	closed        bool
	closedWriting bool
}

func NewByteChannel(capacity int, itemSize int) (ch *ByteChannel) {
	ch = &ByteChannel{
		data:     make([]byte, itemSize*capacity),
		itemSize: int64(itemSize),
		capacity: int64(capacity),
	}

	ch.readCond.L = &ch.mu
	ch.writeCond.L = &ch.mu

	return
}
func (ch *ByteChannel) CopyTo(dst *ByteChannel) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	dst.mu.Lock()
	defer dst.mu.Unlock()

	var i int64

	for i = 0; i < ch.capacity; i++ {
		idx := ch.index(i)

		dst.write(func(b []byte) {
			copy(b, ch.slice(idx))
		})
	}

	dst.awaitingAck = 0

	if ch.length < dst.capacity {
		dst.length = ch.length
	} else {
		dst.length = dst.capacity
	}
}

func (ch *ByteChannel) WriteOrBlock(cb func([]byte)) bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.closedWriting {
		return false
	}

	for !ch.spaceLeft() {
		if ch.closedWriting {
			return false
		}

		// Wait until there is space in the buffer
		ch.writeCond.Wait()
	}

	ch.write(cb)
	return true
}

func (ch *ByteChannel) WriteOrFail(cb func([]byte)) bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.closedWriting || !ch.spaceLeft() {
		return false
	}

	ch.write(cb)
	return true
}

func (ch *ByteChannel) WriteOrReplace(cb func([]byte)) bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.closedWriting {
		return false
	}

	ch.write(cb)
	return true
}

func (ch *ByteChannel) write(cb func([]byte)) {
	idx := ch.index(ch.length)
	cb(ch.slice(idx))

	if ch.spaceLeft() {
		ch.length++
	} else {
		ch.startIdx = ch.index(1)

		if ch.toAck() {
			ch.awaitingAck--
		}
	}

	ch.itemsWritten++
	ch.readCond.Signal()
}

// Wait until there is anything to read
func (ch *ByteChannel) Wait() (unread int64, err error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Wait until there is data in the buffer to read
	for !ch.toRead() && !ch.closed {

		// If writing is closed, there will never be any more to read
		if ch.closedWriting {
			return 0, io.EOF
		}

		ch.readCond.Wait()
	}

	if ch.closed {
		return 0, io.ErrClosedPipe
	}

	return ch.unread(), nil
}

// Wait until something has been read and need to be acknowledged
func (ch *ByteChannel) WaitUntilRead() (read int64, err error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	for !ch.toAck() && !ch.closed {
		ch.writeCond.Wait()
	}

	if ch.closed {
		return 0, io.ErrClosedPipe
	}

	return ch.awaitingAck, nil
}

// Wait until channel is empty
func (ch *ByteChannel) WaitUntilEmpty() (err error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	for !ch.empty() && !ch.closed {
		ch.writeCond.Wait()
	}

	if ch.closed {
		return io.ErrClosedPipe
	}

	return
}

func (ch *ByteChannel) ReadToCallback(cb func([]byte) error, undoOnError bool) (err error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// If there is nothing to read, fail
	if ch.empty() {
		return io.EOF
	}

	err = cb(ch.read())

	if undoOnError && err != nil {
		ch.undoRead()
		ch.readCond.Broadcast()
	} else {
		ch.writeCond.Broadcast()
	}

	return
}

func (ch *ByteChannel) read() []byte {
	idx := ch.index(ch.awaitingAck)
	ch.awaitingAck++
	ch.itemsRead++
	return ch.slice(idx)
}

func (ch *ByteChannel) undoRead() {
	ch.startIdx = ch.index(-1)
	// ch.length++
	ch.awaitingAck--
	ch.itemsRead--
}

func (ch *ByteChannel) Ack() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if !ch.toAck() {
		return
	}

	ch.awaitingAck--
	ch.length--

	if ch.length > 0 {
		ch.startIdx = ch.index(1)
	}

	ch.writeCond.Broadcast()
}

func (ch *ByteChannel) CloseWriting() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if !ch.closedWriting {
		ch.closedWriting = true
		ch.readCond.Signal()
	}
}

func (ch *ByteChannel) Close() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if !ch.closed {
		ch.closed = true
		ch.closedWriting = true
		ch.readCond.Broadcast()
		ch.writeCond.Broadcast()
	}
}

func (ch *ByteChannel) Rewind() (count int64) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	count = ch.awaitingAck
	ch.awaitingAck = 0

	if count > 0 {
		ch.readCond.Broadcast()
	}

	return
}

func (ch *ByteChannel) ToRead() bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.toRead()
}

func (ch *ByteChannel) toRead() bool {
	return ch.unread() > 0
}

func (ch *ByteChannel) ToAck() bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.toAck()
}

func (ch *ByteChannel) toAck() bool {
	return ch.awaitingAck > 0
}

func (ch *ByteChannel) spaceLeft() bool {
	return ch.length < ch.capacity
}

func (ch *ByteChannel) Empty() bool {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.empty()
}

func (ch *ByteChannel) empty() bool {
	return ch.len() <= 0
}

func (ch *ByteChannel) Len() int64 {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.len()
}

func (ch *ByteChannel) len() int64 {
	return ch.length
}

func (ch *ByteChannel) Unread() int64 {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.unread()
}

func (ch *ByteChannel) unread() int64 {
	return ch.length - ch.awaitingAck
}

func (ch *ByteChannel) AwaitingAck() int64 {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.awaitingAck
}

func (ch *ByteChannel) Reset() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.startIdx = 0
	ch.awaitingAck = 0
	ch.length = 0
	ch.writeCond.Broadcast()
}

func (ch *ByteChannel) ItemsWritten() uint64 {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.itemsWritten
}

func (ch *ByteChannel) ItemsRead() uint64 {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.itemsRead
}

func (ch *ByteChannel) slice(index int64) []byte {
	index *= ch.itemSize
	return ch.data[index : index+ch.itemSize]
}

func (ch *ByteChannel) index(index int64) int64 {
	return ch.wrap(ch.startIdx + index)
}

func (ch *ByteChannel) wrap(index int64) int64 {
	return (index + ch.capacity) % ch.capacity
}
