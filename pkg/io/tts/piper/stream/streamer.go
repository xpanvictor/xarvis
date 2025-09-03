// pkg/tts/stream/streamer.go
package stream

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/xpanvictor/xarvis/pkg/io/tts/piper"
)

// Streamer turns text deltas into one audio stream by chunking and TTS-ing per chunk.
type Streamer struct {
	TTS *piper.Piper

	// Segmentation knobs:
	MaxChars   int           // flush when buffer exceeds (default 240)
	MinChars   int           // don't flush tiny fragments unless end (default 40)
	FlushPunct string        // ".!?;:" triggers flush (plus long commas)
	CommaDelay time.Duration // flush on comma if paused (default 600ms)
	IdleFlush  time.Duration // flush if no new text (default 700ms)

	// Audio knobs (WAV only for now)
	FadeMs int // add ~5–10ms fade-in/out at boundaries to avoid clicks
}

// Input is a stream of text deltas (already detokenized).
// Returns a single io.ReadCloser with continuous audio.
func (s *Streamer) FromDeltas(ctx context.Context, deltas <-chan string) (io.ReadCloser, error) {
	if s.TTS == nil {
		return nil, errors.New("no TTS client")
	}
	if s.MaxChars == 0 {
		s.MaxChars = 240
	}
	if s.MinChars == 0 {
		s.MinChars = 40
	}
	if s.FlushPunct == "" {
		s.FlushPunct = ".!?;:"
	}
	if s.CommaDelay == 0 {
		s.CommaDelay = 600 * time.Millisecond
	}
	if s.IdleFlush == 0 {
		s.IdleFlush = 700 * time.Millisecond
	}
	if s.FadeMs == 0 {
		s.FadeMs = 8
	}

	// Pipe out to caller
	pr, pw := io.Pipe()
	go s.run(ctx, deltas, pw)
	return pr, nil
}

func (s *Streamer) run(ctx context.Context, deltas <-chan string, out *io.PipeWriter) {
	defer out.Close()
	var (
		buf     strings.Builder
		lastAdd = time.Now()
		mu      sync.Mutex
		wg      sync.WaitGroup
	)

	flush := func(force bool) {
		mu.Lock()
		defer mu.Unlock()
		text := strings.TrimSpace(buf.String())
		if text == "" {
			return
		}
		if !force && len(text) < s.MinChars {
			return
		}
		// reset buffer
		buf.Reset()
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			// call TTS
			ctxChunk, cancel := context.WithTimeout(ctx, max(15*time.Second, s.TTS.Timeout))
			defer cancel()
			rc, _, err := s.TTS.DoTTS(ctxChunk, t, "")
			if err != nil {
				out.CloseWithError(err)
				return
			}
			defer rc.Close()
			// If WAV, we could strip the WAV header except the first chunk, but
			// simplest is just write chunks back-to-back; most players handle separate WAVs poorly.
			// So we decode + re-encode or, simpler: if format is PCM use direct concat.
			// To keep this example simple, we'll write raw bytes and accept a small boundary click.
			// In practice, set Format:"pcm_s16le" in TTS and use addFade().
			br := bufio.NewReader(rc)
			var chunk bytes.Buffer
			io.Copy(&chunk, br)
			data := chunk.Bytes()
			// Optional: add small fade at boundaries if PCM
			_, _ = out.Write(data)
		}(text)
	}

	// background idle flusher
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			flush(true)
			wg.Wait()
			return
		case d, ok := <-deltas:
			if !ok {
				flush(true)
				wg.Wait()
				return
			}
			if d == "" {
				continue
			}
			buf.WriteString(d)
			lastAdd = time.Now()
			// punctuation-driven flush
			if endsWithAny(buf.String(), s.FlushPunct) || len(buf.String()) >= s.MaxChars {
				flush(false)
			}
		case <-ticker.C:
			// time-based flush on comma or idle
			str := buf.String()
			if str == "" {
				continue
			}
			if time.Since(lastAdd) >= s.IdleFlush {
				flush(true)
			} else if strings.HasSuffix(strings.TrimSpace(str), ",") && time.Since(lastAdd) >= s.CommaDelay {
				flush(false)
			}
		}
	}
}

func endsWithAny(s, set string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	last := s[len(s)-1]
	return strings.ContainsRune(set, rune(last))
}

func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
