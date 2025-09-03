package pipeline

import (
	"context"
	"io"
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
	// 1) Channel for text deltas to the streamer. Buffered to reduce backpressure stalls.
	wordCh := make(chan string, 64)

	// 2) Start TTS streamer *before* feeding deltas, so audio reader is ready.
	audioRC, err := p.str.FromDeltas(ctx, wordCh)
	if err != nil {
		return err
	}
	defer audioRC.Close()

	// 3) Stream audio bytes out as they arrive (no whole-clip buffering).
	//    If your pub can take arbitrary sized frames, just chunk read into 8–32 KB.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		const chunk = 16 * 1024
		buf := make([]byte, chunk)
		for {
			n, rerr := audioRC.Read(buf)
			if n > 0 {
				// publish this audio slice
				_ = p.pub.SendAudioFrame(ctx, userID, sessionID, buf[:n])
			}
			if rerr != nil {
				if rerr != io.EOF {
					// optionally log rerr
				}
				return
			}
		}
	}()

	// 4) Pump LLM deltas -> publish text + forward to TTS.
	//    Close wordCh when upstream closes so streamer can finish.
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
			for _, out := range outputs {
				if msg := out.Msg; msg != nil && msg.Content != "" {
					// publish text delta immediately
					// todo: seq monitoring
					_ = p.pub.SendTextDelta(ctx, userID, sessionID, 0, msg.Content)

					// forward to TTS
					select {
					case wordCh <- msg.Content:
					case <-ctx.Done():
						break pumpLoop
					}
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
	case <-time.After(10 * time.Second):
		// optional timeout to avoid hanging forever
	}

	return nil
}
