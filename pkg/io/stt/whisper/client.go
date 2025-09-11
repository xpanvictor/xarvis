package whisper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"time"

	"github.com/xpanvictor/xarvis/pkg/Logger"
	audioring "github.com/xpanvictor/xarvis/pkg/io/stt/audioRing"
)

// TranscriptionResponse represents the response from Whisper STT service
type TranscriptionResponse struct {
	Text        string                 `json:"text"`
	Language    string                 `json:"language"`
	Segments    []TranscriptionSegment `json:"segments,omitempty"`
	GeneratedAt time.Time
}

// TranscriptionSegment represents a timed segment of transcription
type TranscriptionSegment struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
	ID    int     `json:"id"`
}

// WhisperClient handles communication with Whisper STT service
type WhisperClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *Logger.Logger
}

// NewWhisperClient creates a new Whisper client
func NewWhisperClient(baseURL string, logger *Logger.Logger) *WhisperClient {
	return &WhisperClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// TranscribeAudio sends audio frames to Whisper service and returns transcription
func (w *WhisperClient) TranscribeAudio(ctx context.Context, audioFrames []audioring.AudioInput) (*TranscriptionResponse, error) {
	if len(audioFrames) == 0 {
		return nil, fmt.Errorf("no audio frames provided")
	}

	// Convert audio frames to WAV format
	wavData, err := w.audioFramesToWAV(audioFrames)
	if err != nil {
		return nil, fmt.Errorf("failed to convert audio to WAV: %w", err)
	}

	// Create multipart form data
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add audio file
	part, err := writer.CreateFormFile("audio_file", "audio.wav")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(wavData); err != nil {
		return nil, fmt.Errorf("failed to write audio data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request with query parameters
	initialPrompt := url.QueryEscape("take note of word: xarvis")
	requestURL := fmt.Sprintf("%s/asr?encode=true&task=transcribe&language=en&output=json&initial_prompt=%s",
		w.baseURL, initialPrompt)
	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Send request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read the full response body for debugging
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		w.logger.Errorf("Whisper service error (status %d): %s", resp.StatusCode, string(responseBody))
		return nil, fmt.Errorf("whisper service returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Check if response is empty
	if len(responseBody) == 0 {
		w.logger.Errorf("Whisper service returned empty response")
		return nil, fmt.Errorf("whisper service returned empty response")
	}

	// Log the raw response for debugging
	w.logger.Infof("Raw Whisper response (length=%d): %q", len(responseBody), string(responseBody))

	// Try to parse as JSON first
	var transcription TranscriptionResponse
	if err := json.Unmarshal(responseBody, &transcription); err != nil {
		w.logger.Errorf("Failed to decode JSON response, raw body: %q", string(responseBody))

		// Check if it's a plain text response
		responseText := string(responseBody)
		if responseText != "" {
			w.logger.Infof("Treating response as plain text transcription: %s", responseText)
			return &TranscriptionResponse{
				Text:        responseText,
				Language:    "en", // default
				GeneratedAt: time.Now(),
			}, nil
		}

		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	w.logger.Debugf("Whisper transcription: %s (language: %s)", transcription.Text, transcription.Language)

	return &transcription, nil
}

// audioFramesToWAV converts audio frames to WAV format
func (w *WhisperClient) audioFramesToWAV(audioFrames []audioring.AudioInput) ([]byte, error) {
	if len(audioFrames) == 0 {
		return nil, fmt.Errorf("no audio frames")
	}

	// Use the first frame's sample rate for WAV header
	sampleRate := audioFrames[0].SampleRate
	if sampleRate == 0 {
		sampleRate = 44100 // Default fallback
	}

	// Calculate total audio data size
	totalDataSize := 0
	for _, frame := range audioFrames {
		totalDataSize += len(frame.Data)
	}

	// WAV header constants
	const (
		numChannels   = 1  // Mono
		bitsPerSample = 16 // 16-bit PCM
	)

	// Calculate WAV file size
	byteRate := sampleRate * numChannels * bitsPerSample / 8
	blockAlign := numChannels * bitsPerSample / 8
	wavSize := 44 + totalDataSize // 44 bytes header + data

	// Create WAV header
	header := make([]byte, 44)

	// RIFF chunk descriptor
	copy(header[0:4], "RIFF")
	writeUint32LE(header[4:8], uint32(wavSize-8))
	copy(header[8:12], "WAVE")

	// fmt sub-chunk
	copy(header[12:16], "fmt ")
	writeUint32LE(header[16:20], 16) // PCM format chunk size
	writeUint16LE(header[20:22], 1)  // PCM format
	writeUint16LE(header[22:24], uint16(numChannels))
	writeUint32LE(header[24:28], uint32(sampleRate))
	writeUint32LE(header[28:32], uint32(byteRate))
	writeUint16LE(header[32:34], uint16(blockAlign))
	writeUint16LE(header[34:36], uint16(bitsPerSample))

	// data sub-chunk
	copy(header[36:40], "data")
	writeUint32LE(header[40:44], uint32(totalDataSize))

	// Combine header and audio data
	wavData := make([]byte, 0, wavSize)
	wavData = append(wavData, header...)

	// Append all audio frame data
	for _, frame := range audioFrames {
		wavData = append(wavData, frame.Data...)
	}

	return wavData, nil
}

// Helper functions for writing little-endian bytes
func writeUint32LE(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

func writeUint16LE(b []byte, v uint16) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}
