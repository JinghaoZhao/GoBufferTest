package controller

import (
	"errors"
	"fmt"
	"sync"
)

// Buffer holds incoming DL packets during the UE Idle state
type Buffer struct {
	mutex sync.Mutex

	// Using a circular buffer for packets. If head <= tail, then the useful
	// data is in the interval [head, tail[. If tail < head, then
	// the useful data is the union of [head, len[ and [0, tail[.
	// In order to avoid ambiguity when head = tail, we always leave
	// an unused byte in the buffer.
	// Each packet prepend a 2-byte header to indicate its length
	data       []byte
	head, tail int
	count      int
}

const (
	minSize    = 2048
	cutoffSize = 128 * 1024
)

var (
	// ErrPacketTooBig is returned when the incoming packet is larger than 65536 bytes.
	ErrPacketTooBig = errors.New("packet too big")
)

// NewBuffer creates a new packet buffer.
func NewBuffer() *Buffer {
	return &Buffer{}
}

// available returns true if the buffer is large enough to fit a packet
// of the given size, taking 2-byte length header overhead into account.
func (b *Buffer) available(size int) bool {
	available := b.head - b.tail
	if available <= 0 {
		available += len(b.data)
	}
	// we interpret head=tail as empty, so always keep a byte free
	if size+2+1 > available {
		return false
	}

	return true
}

// grow increases the size of the buffer.  If it returns nil, then the
// buffer has been grown. It returns ErrFull if hits a limit.
func (b *Buffer) grow() error {
	var newsize int
	if len(b.data) < cutoffSize {
		newsize = 2 * len(b.data)
	} else {
		newsize = 5 * len(b.data) / 4
	}
	if newsize < minSize {
		newsize = minSize
	}

	newdata := make([]byte, newsize)

	var n int
	if b.head <= b.tail {
		// data was contiguous
		n = copy(newdata, b.data[b.head:b.tail])
	} else {
		// data was noncontiguous
		n = copy(newdata, b.data[b.head:])
		n += copy(newdata[n:], b.data[:b.tail])
	}
	b.head = 0
	b.tail = n
	b.data = newdata

	return nil
}

// Write appends a copy of the packet data to the buffer.
// Returns ErrFull if the buffer is full
// Returns ErrPacketTooBig if the packet size exceeds 65536 bytes
func (b *Buffer) Write(packet []byte) (int, error) {
	if len(packet) >= 0x10000 {
		return 0, ErrPacketTooBig
	}

	b.mutex.Lock()

	// grow the buffer until the packet fits
	for !b.available(len(packet)) {
		err := b.grow()
		if err != nil {
			b.mutex.Unlock()
			return 0, err
		}
	}

	// store the length of the packet
	b.data[b.tail] = uint8(len(packet) >> 8)
	b.tail++
	if b.tail >= len(b.data) {
		b.tail = 0
	}
	b.data[b.tail] = uint8(len(packet))
	b.tail++
	if b.tail >= len(b.data) {
		b.tail = 0
	}

	// store the packet
	n := copy(b.data[b.tail:], packet)
	b.tail += n
	if b.tail >= len(b.data) {
		// we reached the end, wrap around
		m := copy(b.data, packet[n:])
		b.tail = m
	}
	b.count++
	b.mutex.Unlock()

	return len(packet), nil
}

// Read populates the given byte slice, returning the number of bytes read.
// If return 0, the buffer is empty
// Returns io.ErrShortBuffer is the given packet is too small to copy
func (b *Buffer) Read(packet []byte) (n int, err error) {
	b.mutex.Lock()

	if b.head != b.tail {
		// decode the packet size
		n1 := b.data[b.head]
		b.head++
		if b.head >= len(b.data) {
			b.head = 0
		}
		n2 := b.data[b.head]
		b.head++
		if b.head >= len(b.data) {
			b.head = 0
		}
		count := int((uint16(n1) << 8) | uint16(n2))

		copied := count
		// check if the packet is large enough to hold read data
		if len(packet) < copied {
			errMsg := fmt.Sprintf("short buffer, at least %d is needed", count)
			return 0, errors.New(errMsg)
		}

		// copy the data
		if b.head+copied < len(b.data) {
			copy(packet, b.data[b.head:b.head+copied])
		} else {
			k := copy(packet, b.data[b.head:])
			copy(packet[k:], b.data[:copied-k])
		}

		// advance head
		b.head += count
		if b.head >= len(b.data) {
			b.head -= len(b.data)
		}

		if b.head == b.tail {
			// the buffer is empty, reset to beginning
			// in order to improve cache locality.
			b.head = 0
			b.tail = 0
		}

		b.count--

		b.mutex.Unlock()

		return copied, nil
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
	size := b.tail - b.head
	if size < 0 {
		size += len(b.data)
	}
	return size
}
