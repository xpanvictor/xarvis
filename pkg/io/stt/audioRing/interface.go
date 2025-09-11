package audioring

import (
	"encoding/binary"
	"time"
)

type AudioInput struct {
	Data       []byte
	Timestamp  time.Time
	SampleRate int32
	Channels   int16
}

func (a *AudioInput) MarshalBinary() ([]byte, error) {
	// Create a buffer to hold the serialized data
	// Format: timestamp(8) + sampleRate(4) + channels(2) + dataLen(4) + data
	buf := make([]byte, 8+4+2+4+len(a.Data))

	offset := 0
	// Serialize timestamp (Unix nano)
	binary.LittleEndian.PutUint64(buf[offset:], uint64(a.Timestamp.UnixNano()))
	offset += 8

	// Serialize sample rate
	binary.LittleEndian.PutUint32(buf[offset:], uint32(a.SampleRate))
	offset += 4

	// Serialize channels
	binary.LittleEndian.PutUint16(buf[offset:], uint16(a.Channels))
	offset += 2

	// Serialize data length
	binary.LittleEndian.PutUint32(buf[offset:], uint32(len(a.Data)))
	offset += 4

	// Copy audio data
	copy(buf[offset:], a.Data)

	return buf, nil
}

func (a *AudioInput) UnmarshalBinary(data []byte) error {
	if len(data) < 18 { // minimum size: 8+4+2+4
		return nil
	}

	offset := 0

	// Deserialize timestamp
	timestamp := int64(binary.LittleEndian.Uint64(data[offset:]))
	a.Timestamp = time.Unix(0, timestamp)
	offset += 8

	// Deserialize sample rate
	a.SampleRate = int32(binary.LittleEndian.Uint32(data[offset:]))
	offset += 4

	// Deserialize channels
	a.Channels = int16(binary.LittleEndian.Uint16(data[offset:]))
	offset += 2

	// Deserialize data length
	dataLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4

	// Extract audio data
	if len(data[offset:]) >= int(dataLen) {
		a.Data = make([]byte, dataLen)
		copy(a.Data, data[offset:offset+int(dataLen)])
	}

	return nil
}

type AudioRingBuffer interface {
	Enqueue(audioSlice AudioInput) error
	Dequeue() (AudioInput, bool)
	PeekN(n int32) []AudioInput
	Len() int
	Capacity() int
	Flush(ch chan<- AudioInput) error
}
