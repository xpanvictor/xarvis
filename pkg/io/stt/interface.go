package stt

import (
	"time"

	"github.com/google/uuid"
)

type AudioInput struct {
	AudioStream any // don't know audio format yet
	Period      time.Duration
	StartedAt   time.Time
	ID          uuid.UUID
}

type STTTranscriptEntry struct {
	Content string
	Time    time.Time
}

type STTOutput struct {
	SilenceRatio float32 // 0-1
	Content      string
	// some other meta
	ID             uuid.UUID // uuid from input
	STTGeneratedAt time.Time
	AudioDuration  time.Duration
	Language       string
}

type STTAdapter interface {
	ConvertAudio(in *AudioInput) (STTOutput, error)
	IsAlive() bool
}
