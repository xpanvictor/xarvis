package pipeline

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	xio "github.com/xpanvictor/xarvis/pkg/io"
	"github.com/xpanvictor/xarvis/pkg/io/tts/piper/stream"
)

type Pipeline struct {
	str *stream.Streamer // your segmenting TTS streamer (text deltas -> audio stream)
	pub *xio.Publisher   // abstracts MQTT/WS publishing
}

type GenericEvent struct {
	Key   string
	Value any
}

func New(str *stream.Streamer, pub *xio.Publisher) Pipeline {
	return Pipeline{
		str: str,
		pub: pub,
	}
}

// Broadcast consumes LLM deltas, streams audio frames, and publishes text deltas.
func (p *Pipeline) Broadcast(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
	rc *adapters.ContractResponseChannel, // chan []AdapterOutput (pointer)
	// todo: disable audio
	disableAudio bool,
) error {
	// Channel for text deltas to the streamer. Buffered to reduce backpressure stalls.
	wordCh := make(chan string, 64)

	// Start TTS streamer *before* feeding deltas, so audio reader is ready.
	audioRC, err := p.str.FromDeltas(ctx, wordCh)
	if err != nil {
		return err
	}
	defer audioRC.Close()

	// Setup debug audio saving for troubleshooting
	timestamp := time.Now().Format("20060102_150405")
    // Save debug audio as proper WAV for easy playback
    audioFileName := fmt.Sprintf("%s_%s.wav", sessionID.String(), timestamp)
	debugDir := filepath.Join("out", userID.String())
	debugPath := filepath.Join(debugDir, audioFileName)

	// Create debug directory if it doesn't exist
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		// Log but don't fail, just continue without debug saving
		fmt.Printf("Warning: could not create debug dir %s: %v\n", debugDir, err)
	}

    var debugFile *os.File
    var debugPCMBytes int64 = 0
    if debugDir != "" {
        debugFile, err = os.Create(debugPath)
        if err != nil {
            fmt.Printf("Warning: could not create debug audio file %s: %v\n", debugPath, err)
        } else {
            // Write a placeholder WAV header; we'll fix sizes after writing
            // 16-bit PCM, mono, 22050 Hz to match announced format below
            const (
                sampleRate    = 22050
                channels      = 1
                bitsPerSample = 16
            )
            header := make([]byte, 44)
            // RIFF header
            copy(header[0:4], []byte("RIFF"))
            // ChunkSize (36 + Subchunk2Size) -> placeholder 0 for now
            // We'll backpatch at the end
            // WAVE
            copy(header[8:12], []byte("WAVE"))
            // fmt chunk
            copy(header[12:16], []byte("fmt "))
            // Subchunk1Size (16 for PCM)
            putUint32LE(header[16:20], 16)
            // AudioFormat (1 = PCM)
            putUint16LE(header[20:22], 1)
            // NumChannels
            putUint16LE(header[22:24], uint16(channels))
            // SampleRate
            putUint32LE(header[24:28], uint32(sampleRate))
            // ByteRate = SampleRate * NumChannels * BitsPerSample/8
            byteRate := sampleRate * channels * (bitsPerSample / 8)
            putUint32LE(header[28:32], uint32(byteRate))
            // BlockAlign = NumChannels * BitsPerSample/8
            blockAlign := channels * (bitsPerSample / 8)
            putUint16LE(header[32:34], uint16(blockAlign))
            // BitsPerSample
            putUint16LE(header[34:36], uint16(bitsPerSample))
            // data subchunk
            copy(header[36:40], []byte("data"))
            // Subchunk2Size (data length) -> placeholder 0 for now
            // Write header
            if _, err := debugFile.Write(header); err != nil {
                fmt.Printf("Warning: failed writing WAV header: %v\n", err)
            }
        }
    }

	//  Stream audio bytes out as they arrive (no whole-clip buffering).
	//   just chunk read into 8–32 KB.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if debugFile != nil {
			defer debugFile.Close()
		}

		// Send PCM format metadata at the start
		_ = p.pub.SendEvent(ctx, userID, sessionID, "audio_format", map[string]interface{}{
			"format":        "pcm",
			"sampleRate":    22050,   // Typical Piper sample rate
			"channels":      1,       // Mono
			"bitsPerSample": 16,      // 16-bit signed
			"encoding":      "s16le", // Little-endian
		})

        // Use smaller chunks for smoother streaming (4KB ~= 90ms at 22050Hz)
        // This reduces latency and provides smoother continuous playback
        const chunk = 4 * 1024 // 4KB chunks for better streaming granularity
        buf := make([]byte, chunk)
        audioSeq := 0 // Track audio frame sequence
        for {
            n, rerr := audioRC.Read(buf)
            if n > 0 {
                audioSeq++
                // publish this audio slice with sequence number
                _ = p.pub.SendAudioFrame(ctx, userID, sessionID, audioSeq, buf[:n])

                // Also save to debug file if available
                if debugFile != nil {
                    // Append PCM bytes after the 44-byte WAV header
                    if _, err := debugFile.Write(buf[:n]); err != nil {
                        // Log but don't interrupt streaming
                        fmt.Printf("Warning: failed writing debug audio: %v\n", err)
                    } else {
                        debugPCMBytes += int64(n)
                    }
                }
            }
            if rerr != nil {
                if rerr != io.EOF {
                    // optionally log rerr
                }
                // Send audio_complete event when audio streaming finishes
                _ = p.pub.SendEvent(ctx, userID, sessionID, "audio_complete", map[string]interface{}{
                    "timestamp":   time.Now().Unix(),
                    "totalChunks": audioSeq,
                })
                // Finalize WAV header sizes if we have a debug file
                if debugFile != nil {
                    // Backpatch ChunkSize and Subchunk2Size
                    // ChunkSize = 36 + Subchunk2Size
                    chunkSize := uint32(36 + debugPCMBytes)
                    // Write ChunkSize
                    if _, err := debugFile.Seek(4, 0); err == nil {
                        b := make([]byte, 4)
                        putUint32LE(b, chunkSize)
                        _, _ = debugFile.Write(b)
                    }
                    // Write Subchunk2Size at offset 40
                    if _, err := debugFile.Seek(40, 0); err == nil {
                        b := make([]byte, 4)
                        putUint32LE(b, uint32(debugPCMBytes))
                        _, _ = debugFile.Write(b)
                    }
                }
                return
            }
        }
    }()

	// Pump LLM deltas -> batch per tick, publish once, forward once to TTS.
	// Close wordCh when upstream closes so streamer can finish.
pumpLoop:
	for {
		select {
		case <-ctx.Done():
			break pumpLoop

		case outputs, ok := <-*rc:
			if !ok {
				// upstream closed -> flush and end
				break pumpLoop
			}
			// combine all text in this tick and track highest seq index
			var batch string
			var maxIdx uint
			for _, out := range outputs {
				if msg := out.Msg; msg != nil && msg.Content != "" {
					batch += msg.Content
					if out.Index > maxIdx {
						maxIdx = out.Index
					}
				}
			}
			if batch != "" {
				// publish combined text once with sequence for ordering on client
				_ = p.pub.SendTextDelta(ctx, userID, sessionID, int(maxIdx), batch)
				// forward combined text once to TTS
				select {
				case wordCh <- batch:
				case <-ctx.Done():
					break pumpLoop
				}
			}
		}
	}

	// 5) Signal no more text -> let streamer flush & close.
	close(wordCh)

	// 6) Wait for audio drain.
	doneCh := make(chan struct{})
	go func() { wg.Wait(); close(doneCh) }()

	select {
	case <-doneCh:
		// Send completion event when everything is done
		_ = p.pub.SendEvent(ctx, userID, sessionID, "message_complete", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
		if debugFile != nil {
			// fmt.Printf("Debug: audio saved to %s\n", debugPath)
		}
	case <-time.After(10 * time.Second):
		// optional timeout to avoid hanging forever
		_ = p.pub.SendEvent(ctx, userID, sessionID, "message_complete", map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"timeout":   true,
		})
		if debugFile != nil {
			// fmt.Printf("Debug: audio saved to %s (timeout)\n", debugPath)
		}
	}

    return nil
}

// Helpers to write little-endian integers
func putUint16LE(b []byte, v uint16) {
    b[0] = byte(v)
    b[1] = byte(v >> 8)
}

func putUint32LE(b []byte, v uint32) {
    b[0] = byte(v)
    b[1] = byte(v >> 8)
    b[2] = byte(v >> 16)
    b[3] = byte(v >> 24)
}

func (p *Pipeline) SendEvent(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
	event GenericEvent,
) {
	p.pub.SendEvent(ctx, userID, sessionID, event.Key, event.Value)
}
