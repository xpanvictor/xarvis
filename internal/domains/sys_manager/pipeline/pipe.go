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
	audioFileName := fmt.Sprintf("%s_%s.mp3", sessionID.String(), timestamp)
	debugDir := filepath.Join("out", userID.String())
	debugPath := filepath.Join(debugDir, audioFileName)

	// Create debug directory if it doesn't exist
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		// Log but don't fail, just continue without debug saving
		fmt.Printf("Warning: could not create debug dir %s: %v\n", debugDir, err)
	}

	var debugFile *os.File
	if debugDir != "" {
		debugFile, err = os.Create(debugPath)
		if err != nil {
			fmt.Printf("Warning: could not create debug audio file %s: %v\n", debugPath, err)
		} else {
			// fmt.Printf("Debug: saving complete audio to %s\n", debugPath)
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
		const chunk = 16 * 1024
		buf := make([]byte, chunk)
		for {
			n, rerr := audioRC.Read(buf)
			if n > 0 {
				// publish this audio slice
				_ = p.pub.SendAudioFrame(ctx, userID, sessionID, buf[:n])

				// Also save to debug file if available
				if debugFile != nil {
					debugFile.Write(buf[:n])
				}
			}
			if rerr != nil {
				if rerr != io.EOF {
					// optionally log rerr
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
