package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffer(t *testing.T) {
	assert := assert.New(t)

	buffer := NewBuffer()
	packet := make([]byte, 4)

	// Write once
	n, err := buffer.Write([]byte{0, 1})
	assert.NoError(err)
	assert.Equal(2, n)

	// Read once
	n, err = buffer.Read(packet)
	assert.NoError(err)
	assert.Equal(2, n)
	assert.Equal([]byte{0, 1}, packet[:n])

	// Write twice
	n, err = buffer.Write([]byte{2, 3, 4})
	assert.NoError(err)
	assert.Equal(3, n)

	n, err = buffer.Write([]byte{5, 6, 7})
	assert.NoError(err)
	assert.Equal(3, n)

	// Check the buffer count
	count := buffer.Count()
	assert.Equal(2, count)

	// Check the buffer size, two packets with 2-byte length header + 3-byte payload
	size := buffer.Size()
	assert.Equal(3+3, size)

	// Read twice
	n, err = buffer.Read(packet)
	assert.NoError(err)
	assert.Equal(3, n)
	assert.Equal([]byte{2, 3, 4}, packet[:n])

	n, err = buffer.Read(packet)
	assert.NoError(err)
	assert.Equal(3, n)
	assert.Equal([]byte{5, 6, 7}, packet[:n])

	// Test Read an empty buffer
	n, err = buffer.Read(packet)
	assert.NoError(err)
	assert.Equal(0, n)
}

func benchmarkBufferWR(b *testing.B, size int64, write bool, grow int) { // nolint:unparam
	buffer := NewBuffer()
	packet := make([]byte, size)

	// Grow the buffer first
	pad := make([]byte, 1022)
	for buffer.Size() < grow {
		_, err := buffer.Write(pad)
		if err != nil {
			b.Fatalf("Write: %v", err)
		}
	}
	for buffer.Size() > 0 {
		_, err := buffer.Read(pad)
		if err != nil {
			b.Fatalf("Write: %v", err)
		}
	}

	if write {
		_, err := buffer.Write(packet)
		if err != nil {
			b.Fatalf("Write: %v", err)
		}
	}

	b.SetBytes(size)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := buffer.Write(packet)
		if err != nil {
			b.Fatalf("Write: %v", err)
		}
		_, err = buffer.Read(packet)
		if err != nil {
			b.Fatalf("Read: %v", err)
		}
	}
}

// In this benchmark, the buffer is often empty, which is hopefully
// typical of real usage.
func BenchmarkBufferWR14(b *testing.B) {
	benchmarkBufferWR(b, 14, false, 10*1024*1024)
}

func BenchmarkBufferWR140(b *testing.B) {
	benchmarkBufferWR(b, 140, false, 10*1024*1024)
}

func BenchmarkBufferWR1400(b *testing.B) {
	benchmarkBufferWR(b, 1400, false, 10*1024*1024)
}

// Here, the buffer never becomes empty, which forces wraparound
func BenchmarkBufferWWR14(b *testing.B) {
	benchmarkBufferWR(b, 14, true, 10*1024*1024)
}

func BenchmarkBufferWWR140(b *testing.B) {
	benchmarkBufferWR(b, 140, true, 10*1024*1024)
}

func BenchmarkBufferWWR1400(b *testing.B) {
	benchmarkBufferWR(b, 1400, true, 10*1024*1024)
}
