package vad

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/xpanvictor/xarvis/pkg/Logger"
	audioring "github.com/xpanvictor/xarvis/pkg/io/stt/audioRing"
)

// VADSegment represents a voice activity segment from Silero API
type VADSegment struct {
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Confidence float64 `json:"confidence"`
}

// SileroAPIResponse represents the response from Silero VAD service
type SileroAPIResponse struct {
	HasVoice         bool         `json:"has_voice"`
	Confidence       float32      `json:"confidence"`
	Segments         []VADSegment `json:"segments"`
	ProcessingTimeMs float64      `json:"processing_time_ms"`
	AudioDurationMs  float64      `json:"audio_duration_ms"`
}

// SileroVAD implements VAD using Silero VAD model via HTTP API
type SileroVAD struct {
	config     VADConfig
	logger     *Logger.Logger
	mutex      sync.Mutex
	closed     bool
	httpClient *http.Client
	serviceURL string // URL of the Silero VAD service
}

// NewSileroVAD creates a new Silero VAD instance
func NewSileroVAD(config VADConfig, logger *Logger.Logger) *SileroVAD {
	return &SileroVAD{
		config:     config,
		logger:     logger,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		serviceURL: "http://localhost:8001", // Default local service URL
	}
}

// NewSileroVADWithURL creates a new Silero VAD instance with custom service URL
func NewSileroVADWithURL(config VADConfig, logger *Logger.Logger, serviceURL string) *SileroVAD {
	return &SileroVAD{
		config:     config,
		logger:     logger,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		serviceURL: serviceURL,
	}
}

// DetectVoice analyzes audio data using Silero VAD service
func (s *SileroVAD) DetectVoice(ctx context.Context, audio audioring.AudioInput) (VADResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.closed {
		return VADResult{}, fmt.Errorf("VAD is closed")
	}

	// Convert audio data to the format expected by Silero VAD
	if len(audio.Data) == 0 {
		return VADResult{HasVoice: false, Confidence: 0.0}, nil
	}

	// Check if we have sufficient audio data (at least 100ms)
	minSamples := int(s.config.SampleRate * 100 / 1000) // 100ms worth of samples
	actualSamples := len(audio.Data) / 2                // Assuming 16-bit PCM (2 bytes per sample)

	if actualSamples < minSamples {
		return VADResult{HasVoice: false, Confidence: 0.0}, nil
	}

	// Try to call Silero VAD service, fallback to energy-based VAD
	result, err := s.callSileroVADService(ctx, audio)
	if err != nil {
		s.logger.Warnf("Silero VAD service failed, falling back to energy-based VAD: %v", err)
		return s.energyBasedVAD(audio)
	}

	return result, nil
}

// energyBasedVAD implements a simple energy-based voice activity detection
func (s *SileroVAD) energyBasedVAD(audio audioring.AudioInput) (VADResult, error) {
	if len(audio.Data) < 2 {
		return VADResult{HasVoice: false, Confidence: 0.0}, nil
	}

	// Calculate RMS energy of the audio signal
	var sum int64
	sampleCount := len(audio.Data) / 2 // 16-bit samples

	for i := 0; i < len(audio.Data)-1; i += 2 {
		// Convert bytes to 16-bit sample (little endian)
		sample := int16(audio.Data[i]) | (int16(audio.Data[i+1]) << 8)
		sum += int64(sample) * int64(sample)
	}

	if sampleCount == 0 {
		return VADResult{HasVoice: false, Confidence: 0.0}, nil
	}

	rms := float32(sum) / float32(sampleCount)
	energy := float32(rms) / (32768.0 * 32768.0) // Normalize to 0-1 range

	// Simple threshold-based detection
	hasVoice := energy > s.config.Threshold
	confidence := energy / s.config.Threshold
	if confidence > 1.0 {
		confidence = 1.0
	}

	s.logger.Debugf("VAD: energy=%.6f, threshold=%.3f, hasVoice=%v, confidence=%.3f",
		energy, s.config.Threshold, hasVoice, confidence)

	return VADResult{
		HasVoice:   hasVoice,
		Confidence: confidence,
	}, nil
}

// callSileroVADService calls the Silero VAD HTTP service
func (s *SileroVAD) callSileroVADService(ctx context.Context, audio audioring.AudioInput) (VADResult, error) {
	// Create WAV header for the raw PCM data
	wavData, err := s.createWAVFromPCM(audio.Data)
	if err != nil {
		return VADResult{}, fmt.Errorf("failed to create WAV data: %v", err)
	}

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add audio file
	part, err := writer.CreateFormFile("file", "audio.wav")
	if err != nil {
		return VADResult{}, fmt.Errorf("failed to create form file: %v", err)
	}

	if _, err := part.Write(wavData); err != nil {
		return VADResult{}, fmt.Errorf("failed to write audio data: %v", err)
	}

	// Add VAD parameters
	writer.WriteField("threshold", fmt.Sprintf("%.3f", s.config.Threshold))
	writer.WriteField("min_speech_duration_ms", strconv.Itoa(s.config.MinSpeechMs))
	writer.WriteField("min_silence_duration_ms", strconv.Itoa(s.config.MinSilenceMs))
	writer.WriteField("sampling_rate", strconv.Itoa(int(s.config.SampleRate)))

	writer.Close()

	// Create HTTP request
	url := s.serviceURL + "/vad"
	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return VADResult{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Make the request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return VADResult{}, fmt.Errorf("failed to call VAD service: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return VADResult{}, fmt.Errorf("VAD service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var sileroResp SileroAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&sileroResp); err != nil {
		return VADResult{}, fmt.Errorf("failed to decode response: %v", err)
	}

	s.logger.Debugf("Silero VAD: hasVoice=%v, confidence=%.3f, segments=%d, processing_time=%.1fms",
		sileroResp.HasVoice, sileroResp.Confidence, len(sileroResp.Segments), sileroResp.ProcessingTimeMs)

	// Convert response to our format
	result := VADResult{
		HasVoice:   sileroResp.HasVoice,
		Confidence: sileroResp.Confidence,
	}

	// Add timing information from first segment if available
	if len(sileroResp.Segments) > 0 {
		firstSegment := sileroResp.Segments[0]
		// Convert from seconds to sample indices
		result.StartTime = int64(firstSegment.Start * float64(s.config.SampleRate))
		result.EndTime = int64(firstSegment.End * float64(s.config.SampleRate))
	}

	return result, nil
}

// createWAVFromPCM creates a WAV file from raw PCM data
func (s *SileroVAD) createWAVFromPCM(pcmData []byte) ([]byte, error) {
	buffer := &bytes.Buffer{}

	// WAV header
	dataSize := uint32(len(pcmData))
	fileSize := dataSize + 36

	// Write WAV header
	buffer.WriteString("RIFF")
	buffer.Write(uint32ToBytes(fileSize))
	buffer.WriteString("WAVE")
	buffer.WriteString("fmt ")
	buffer.Write(uint32ToBytes(16))                              // PCM format chunk size
	buffer.Write(uint16ToBytes(1))                               // PCM format
	buffer.Write(uint16ToBytes(1))                               // Mono
	buffer.Write(uint32ToBytes(uint32(s.config.SampleRate)))     // Sample rate
	buffer.Write(uint32ToBytes(uint32(s.config.SampleRate * 2))) // Byte rate (sample rate * channels * bytes per sample)
	buffer.Write(uint16ToBytes(2))                               // Block align (channels * bytes per sample)
	buffer.Write(uint16ToBytes(16))                              // Bits per sample
	buffer.WriteString("data")
	buffer.Write(uint32ToBytes(dataSize))

	// Write PCM data
	buffer.Write(pcmData)

	return buffer.Bytes(), nil
}

// Helper functions for WAV header
func uint32ToBytes(val uint32) []byte {
	return []byte{
		byte(val & 0xFF),
		byte((val >> 8) & 0xFF),
		byte((val >> 16) & 0xFF),
		byte((val >> 24) & 0xFF),
	}
}

func uint16ToBytes(val uint16) []byte {
	return []byte{
		byte(val & 0xFF),
		byte((val >> 8) & 0xFF),
	}
}

// getPythonScript returns the Python script for Silero VAD
func (s *SileroVAD) getPythonScript() string {
	return `
import torch
import numpy as np
import json
import sys

# This is a placeholder for Silero VAD integration
# TODO: Implement actual Silero VAD model loading and inference

def detect_voice(audio_data, sample_rate=16000, threshold=0.5):
    """
    Detect voice activity using Silero VAD
    """
    # Placeholder implementation
    return {
        "hasVoice": False,
        "confidence": 0.0
    }

if __name__ == "__main__":
    # Read audio data from stdin or arguments
    # Process with Silero VAD
    # Output JSON result
    result = detect_voice([])
    print(json.dumps(result))
`
}

// Close releases resources
func (s *SileroVAD) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.closed = true
	s.logger.Debugf("Silero VAD closed")
	return nil
}

// DefaultVADConfig returns default VAD configuration optimized for speech
func DefaultVADConfig() VADConfig {
	return VADConfig{
		SampleRate:   16000, // 16kHz is optimal for speech
		Threshold:    0.3,   // Balanced threshold - not too sensitive, not too strict
		MinSpeechMs:  100,   // Minimum 100ms of speech
		MinSilenceMs: 200,   // Minimum 200ms of silence
	}
}
