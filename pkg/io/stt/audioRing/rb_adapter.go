package audioring

import (
	"errors"

	"github.com/smallnest/ringbuffer"
)

type rb_impl struct {
	size int
	rb   *ringbuffer.RingBuffer
}

// Capacity implements AudioRingBuffer.
func (r *rb_impl) Capacity() int {
	return r.size
}

// Dequeue implements AudioRingBuffer.
func (r *rb_impl) Dequeue() (AudioInput, bool) {
	if r.rb.IsEmpty() {
		return AudioInput{}, false
	}

	// Read the size of the next item first (we'll store size as first 4 bytes)
	sizeBytes := make([]byte, 4)
	n, err := r.rb.Read(sizeBytes)
	if err != nil || n != 4 {
		return AudioInput{}, false
	}

	// Convert bytes to size
	size := int(sizeBytes[0]) | int(sizeBytes[1])<<8 | int(sizeBytes[2])<<16 | int(sizeBytes[3])<<24

	// Read the actual data
	data := make([]byte, size)
	n, err = r.rb.Read(data)
	if err != nil || n != size {
		return AudioInput{}, false
	}

	// Unmarshal the data
	var audioInput AudioInput
	err = audioInput.UnmarshalBinary(data)
	if err != nil {
		return AudioInput{}, false
	}

	return audioInput, true
} // Enqueue implements AudioRingBuffer.
func (r *rb_impl) Enqueue(audioSlice AudioInput) error {
	// Marshal the audio input to bytes
	data, err := audioSlice.MarshalBinary()
	if err != nil {
		return err
	}

	// Calculate required space (data size + 4 bytes for size prefix)
	requiredSpace := len(data) + 4

	// If the required space is larger than our entire buffer, that's an error
	if requiredSpace > r.rb.Capacity() {
		return errors.New("audio frame too large for buffer")
	}

	// Make space by removing old data if necessary
	for r.rb.Free() < requiredSpace {
		// Remove one complete audio frame from the front
		if !r.removeOldestFrame() {
			// If we can't remove frames (buffer is corrupted), reset it
			r.rb.Reset()
			break
		}
	}

	// Write the size first (4 bytes, little endian)
	sizeBytes := make([]byte, 4)
	sizeBytes[0] = byte(len(data))
	sizeBytes[1] = byte(len(data) >> 8)
	sizeBytes[2] = byte(len(data) >> 16)
	sizeBytes[3] = byte(len(data) >> 24)

	_, err = r.rb.Write(sizeBytes)
	if err != nil {
		return err
	}

	// Write the actual data
	_, err = r.rb.Write(data)
	return err
}

// removeOldestFrame removes the oldest complete audio frame from the buffer
func (r *rb_impl) removeOldestFrame() bool {
	if r.rb.IsEmpty() {
		return false
	}

	// Read the size of the next frame
	sizeBytes := make([]byte, 4)
	n, err := r.rb.Read(sizeBytes)
	if err != nil || n != 4 {
		return false
	}

	// Convert bytes to size
	size := int(sizeBytes[0]) | int(sizeBytes[1])<<8 | int(sizeBytes[2])<<16 | int(sizeBytes[3])<<24

	// Skip the frame data
	if size > 0 {
		skipData := make([]byte, size)
		n, err := r.rb.Read(skipData)
		if err != nil || n != size {
			return false
		}
	}

	return true
}

// Flush implements AudioRingBuffer.
func (r *rb_impl) Flush(ch chan<- AudioInput) error {
	defer close(ch)

	for !r.rb.IsEmpty() {
		audio, ok := r.Dequeue()
		if !ok {
			break
		}

		select {
		case ch <- audio:
		default:
			// Channel is blocked, return error
			return errors.New("channel blocked during flush")
		}
	}

	return nil
}

// Len implements AudioRingBuffer.
func (r *rb_impl) Len() int {
	return r.rb.Length()
}

// PeekN implements AudioRingBuffer.
func (r *rb_impl) PeekN(n int32) []AudioInput {
	result := make([]AudioInput, 0, n)

	if r.rb.IsEmpty() {
		return result
	}

	// Create a temporary ring buffer to peek without consuming
	tempRB := ringbuffer.New(r.rb.Capacity())

	// Copy current buffer contents to temp buffer
	buf := make([]byte, r.rb.Length())
	tempData := make([]byte, r.rb.Length())
	r.rb.Bytes(buf)
	copy(tempData, buf)
	tempRB.Write(tempData)

	count := int32(0)
	for count < n && !tempRB.IsEmpty() {
		// Read size prefix
		sizeBytes := make([]byte, 4)
		readN, err := tempRB.Read(sizeBytes)
		if err != nil || readN != 4 {
			break
		}

		// Calculate size
		size := int(sizeBytes[0]) | int(sizeBytes[1])<<8 | int(sizeBytes[2])<<16 | int(sizeBytes[3])<<24

		// Read data
		data := make([]byte, size)
		readN, err = tempRB.Read(data)
		if err != nil || readN != size {
			break
		}

		// Unmarshal
		var audioInput AudioInput
		err = audioInput.UnmarshalBinary(data)
		if err != nil {
			break
		}

		result = append(result, audioInput)
		count++
	}

	return result
}

func New(size int) AudioRingBuffer {
	return &rb_impl{
		size: size,
		rb:   ringbuffer.New(size).SetBlocking(false), // Non-blocking for graceful overflow handling
	}
}
