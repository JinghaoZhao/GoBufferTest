package list

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

// Buffer holds incoming DL packets during the UE Idle state
type Buffer struct {
	mutex sync.Mutex

	pktqueue          *list.List
	count, buffersize int
}

type Packet struct {
	payload []byte
}

type Metadata struct {
	Timestamp time.Time
}

var (
	// ErrPacketTooBig is returned when the incoming packet is larger than 65536 bytes.
	ErrPacketTooBig = errors.New("packet too big")
)

// NewBuffer creates a new packet buffer.
func NewBuffer() *Buffer {
	return &Buffer{
		pktqueue: list.New(),
	}
}

// Write appends a copy of the packet data to the buffer.
// Returns ErrFull if the buffer is full
// Returns ErrPacketTooBig if the packet size exceeds 65536 bytes
func (b *Buffer) Write(packet []byte) (int, error) {
	if len(packet) >= 0x10000 {
		return 0, ErrPacketTooBig
	}

	b.mutex.Lock()

	pkt := &Packet{
		payload: packet,
	}

	b.pktqueue.PushBack(pkt)
	pktlen := len(packet)
	b.buffersize += pktlen
	b.count++
	b.mutex.Unlock()

	return pktlen, nil
}

// Read populates the given byte slice, returning the number of bytes read.
// If return 0, the buffer is empty
// Returns io.ErrShortBuffer is the given packet is too small to copy
func (b *Buffer) Read(packet []byte) (n int, err error) {
	b.mutex.Lock()

	if b.pktqueue.Len() > 0 {
		pktpt := b.pktqueue.Front()
		pkt := pktpt.Value.(*Packet)
		copy(packet, pkt.payload)

		b.count--
		pktlen := len(pkt.payload)
		b.buffersize -= pktlen

		b.pktqueue.Remove(pktpt)

		b.mutex.Unlock()
		return pktlen, nil
	}

	b.mutex.Unlock()
	return 0, nil
}

// Count returns the number of packets in the buffer.
func (b *Buffer) Count() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.count
}

// Size returns the total byte size of packets in the buffer, including
// a small amount of extra length header.
func (b *Buffer) Size() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.size()
}

func (b *Buffer) size() int {
	return b.buffersize
}
