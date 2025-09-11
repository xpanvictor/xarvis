package audioring

import (
	"testing"
	"time"
)

func TestAudioRingBuffer(t *testing.T) {
	// Create a new ring buffer
	buffer := New(1024)

	// Test basic properties
	if buffer.Capacity() != 1024 {
		t.Errorf("Expected capacity 1024, got %d", buffer.Capacity())
	}

	if buffer.Len() != 0 {
		t.Errorf("Expected empty buffer, got length %d", buffer.Len())
	}

	// Create test audio input
	audio1 := AudioInput{
		Data:       []byte{1, 2, 3, 4, 5},
		Timestamp:  time.Now(),
		SampleRate: 44100,
		Channels:   2,
	}

	// Test enqueue
	err := buffer.Enqueue(audio1)
	if err != nil {
		t.Errorf("Failed to enqueue: %v", err)
	}

	if buffer.Len() == 0 {
		t.Error("Buffer should not be empty after enqueue")
	}

	// Test dequeue
	dequeued, ok := buffer.Dequeue()
	if !ok {
		t.Error("Failed to dequeue")
	}

	// Verify dequeued data
	if len(dequeued.Data) != len(audio1.Data) {
		t.Errorf("Expected data length %d, got %d", len(audio1.Data), len(dequeued.Data))
	}

	for i, b := range dequeued.Data {
		if b != audio1.Data[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, audio1.Data[i], b)
		}
	}

	if dequeued.SampleRate != audio1.SampleRate {
		t.Errorf("Expected sample rate %d, got %d", audio1.SampleRate, dequeued.SampleRate)
	}

	if dequeued.Channels != audio1.Channels {
		t.Errorf("Expected channels %d, got %d", audio1.Channels, dequeued.Channels)
	}
}

func TestAudioRingBufferMultiple(t *testing.T) {
	buffer := New(1024)

	// Enqueue multiple items
	for i := 0; i < 3; i++ {
		audio := AudioInput{
			Data:       []byte{byte(i), byte(i + 1), byte(i + 2)},
			Timestamp:  time.Now().Add(time.Duration(i) * time.Millisecond),
			SampleRate: 44100,
			Channels:   2,
		}
		err := buffer.Enqueue(audio)
		if err != nil {
			t.Errorf("Failed to enqueue item %d: %v", i, err)
		}
	}

	// Test PeekN
	peeked := buffer.PeekN(2)
	if len(peeked) != 2 {
		t.Errorf("Expected 2 peeked items, got %d", len(peeked))
	}

	// Buffer should still have all items after peek
	if buffer.Len() == 0 {
		t.Error("Buffer should not be empty after peek")
	}

	// Test flush
	ch := make(chan AudioInput, 10)
	err := buffer.Flush(ch)
	if err != nil {
		t.Errorf("Failed to flush: %v", err)
	}

	// Check flushed items
	flushedCount := 0
	for range ch {
		flushedCount++
	}

	if flushedCount != 3 {
		t.Errorf("Expected 3 flushed items, got %d", flushedCount)
	}

	// Buffer should be empty after flush
	if buffer.Len() != 0 {
		t.Errorf("Buffer should be empty after flush, got length %d", buffer.Len())
	}
}

func TestAudioInputSerialization(t *testing.T) {
	original := AudioInput{
		Data:       []byte{10, 20, 30, 40, 50},
		Timestamp:  time.Now(),
		SampleRate: 48000,
		Channels:   1,
	}

	// Test marshaling
	data, err := original.MarshalBinary()
	if err != nil {
		t.Errorf("Failed to marshal: %v", err)
	}

	// Test unmarshaling
	var restored AudioInput
	err = restored.UnmarshalBinary(data)
	if err != nil {
		t.Errorf("Failed to unmarshal: %v", err)
	}

	// Verify restored data
	if len(restored.Data) != len(original.Data) {
		t.Errorf("Expected data length %d, got %d", len(original.Data), len(restored.Data))
	}

	for i, b := range restored.Data {
		if b != original.Data[i] {
			t.Errorf("Data mismatch at index %d: expected %d, got %d", i, original.Data[i], b)
		}
	}

	if restored.SampleRate != original.SampleRate {
		t.Errorf("Expected sample rate %d, got %d", original.SampleRate, restored.SampleRate)
	}

	if restored.Channels != original.Channels {
		t.Errorf("Expected channels %d, got %d", original.Channels, restored.Channels)
	}

	// Timestamps should be very close (within a small margin due to precision)
	timeDiff := restored.Timestamp.Sub(original.Timestamp)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > time.Microsecond {
		t.Errorf("Timestamp difference too large: %v", timeDiff)
	}
}
