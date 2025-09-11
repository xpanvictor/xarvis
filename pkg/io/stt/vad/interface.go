package vad

import (
	"context"

	audioring "github.com/xpanvictor/xarvis/pkg/io/stt/audioRing"
)

// VADResult represents the result of voice activity detection
type VADResult struct {
	HasVoice   bool    `json:"hasVoice"`
	Confidence float32 `json:"confidence"`
	StartTime  int64   `json:"startTime,omitempty"` // Start time in samples
	EndTime    int64   `json:"endTime,omitempty"`   // End time in samples
}

// VAD interface for voice activity detection
type VAD interface {
	// DetectVoice analyzes audio data and returns VAD result
	DetectVoice(ctx context.Context, audio audioring.AudioInput) (VADResult, error)

	// Close releases any resources
	Close() error
}

// VADConfig contains configuration for VAD
type VADConfig struct {
	SampleRate   int32   `json:"sampleRate"`   // Expected sample rate (e.g., 16000)
	Threshold    float32 `json:"threshold"`    // Voice detection threshold (0.0-1.0)
	MinSpeechMs  int     `json:"minSpeechMs"`  // Minimum speech duration in milliseconds
	MinSilenceMs int     `json:"minSilenceMs"` // Minimum silence duration in milliseconds
}
