// pkg/tts/stream/streamer.go
package stream

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"log"
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
	ForceFlush time.Duration // flush even if text keeps coming (default 800ms)

	// Audio knobs (WAV only for now)
	FadeMs int // add ~5–10ms fade-in/out at boundaries to avoid clicks
}

func New(tts *piper.Piper) Streamer {
	return Streamer{
		TTS: tts,
		// More aggressive defaults for better streaming
		MaxChars:   120, // Reduced from 240
		MinChars:   15,  // Reduced from 40
		FlushPunct: ".!?;:",
		CommaDelay: 300 * time.Millisecond, // Reduced from 600ms
		IdleFlush:  5 * time.Second,
		ForceFlush: 8 * time.Second, // increased from 800ms
		FadeMs:     8,
	}
}

// Input is a stream of text deltas (already detokenized).
// Returns a single io.ReadCloser with continuous audio.
func (s *Streamer) FromDeltas(ctx context.Context, deltas <-chan string) (io.ReadCloser, error) {
	if s.TTS == nil {
		return nil, errors.New("no TTS client")
	}
	if s.MaxChars == 0 {
		s.MaxChars = 80 // Reduced from 240
	}
	if s.MinChars == 0 {
		s.MinChars = 15 // Reduced from 40
	}
	if s.FlushPunct == "" {
		s.FlushPunct = ".!?;:," // Added comma
	}
	if s.CommaDelay == 0 {
		s.CommaDelay = 300 * time.Millisecond // Reduced from 600ms
	}
	if s.IdleFlush == 0 {
		s.IdleFlush = 5 * time.Second // Reduced from 700ms
	}
	if s.ForceFlush == 0 {
		s.ForceFlush = 8 * time.Second // Reduced from 800ms
	}
	if s.FadeMs == 0 {
		s.FadeMs = 8
	}

	// Pipe out to caller
	pr, pw := io.Pipe()
	log.Printf("streamer: FromDeltas start (Max=%d Min=%d Idle=%s Force=%s)", s.MaxChars, s.MinChars, s.IdleFlush, s.ForceFlush)
	go s.run(ctx, deltas, pw)
	return pr, nil
}

func (s *Streamer) run(ctx context.Context, deltas <-chan string, out *io.PipeWriter) {
	defer out.Close()
	var (
		buf       strings.Builder
		lastAdd   = time.Now()
		lastFlush = time.Now()
		mu        sync.Mutex
		wg        sync.WaitGroup
	)

	flush := func(force bool, reason string) {
		mu.Lock()
		defer mu.Unlock()
		text := strings.TrimSpace(buf.String())
		if text == "" {
			return
		}
		// force should require proper handling
		if force && len(text) < s.MinChars {
			return
		}
		// reset buffer
		buf.Reset()
		// log.Printf("streamer: flush len=%d force=%v reason=%s", len(text), force, reason)
		wg.Add(1)
		go func(t string) {
			defer wg.Done()

			// call TTS
			ctxChunk, cancel := context.WithTimeout(ctx, max(50*time.Second, s.TTS.Timeout))

			log.Printf("streamer: calling DoTTS on `%v` due to [%v]:[%t]", t, reason, force)
			rc, ct, err := s.TTS.DoTTS(ctxChunk, t, "")
			if err != nil {
				log.Printf("streamer: TTS error: %v", err)
				cancel()
				out.CloseWithError(err)
				return
			}

			// Read all audio data first while context is still valid
			audioData, err := io.ReadAll(rc)
			rc.Close()
			cancel() // Cancel the timeout context since we're done with the request
			if err != nil {
				log.Printf("streamer: failed to read TTS audio: %v (main ctx done: %v, chunk ctx done: %v)",
					err,
					ctx.Err() != nil,
					ctxChunk.Err() != nil)
				out.CloseWithError(err)
				return
			}

			// Now convert to MP3 without any context dependencies
			mp3Audio, err := ConvertAudioToMP3(bytes.NewReader(audioData), ct)
			if err != nil {
				log.Printf("streamer: conversion to mp3 error: %v", err)
				out.CloseWithError(err)
				return
			}
			// If WAV, we could strip the WAV header except the first chunk, but
			// simplest is just write chunks back-to-back; most players handle separate WAVs poorly.
			// So we decode + re-encode or, simpler: if format is PCM use direct concat.
			// To keep this example simple, we'll write raw bytes and accept a small boundary click.
			// In practice, set Format:"pcm_s16le" in TTS and use addFade().
			br := bufio.NewReader(mp3Audio)
			var chunk bytes.Buffer
			io.Copy(&chunk, br)
			data := chunk.Bytes()
			// Optional: add small fade at boundaries if PCM
			_, _ = out.Write(data)
		}(text)
		lastFlush = time.Now()
	}

	// background idle flusher
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			flush(true, "ctx-done")
			wg.Wait()
			return
		case d, ok := <-deltas:
			if !ok {
				flush(true, "deltas-closed")
				wg.Wait()
				return
			}
			if d == "" {
				continue
			}
			buf.WriteString(d)
			lastAdd = time.Now()
			// punctuation-driven flush
			currentText := buf.String()
			if endsWithAny(currentText, s.FlushPunct) || len(currentText) >= s.MaxChars {
				flush(false, "punct-or-max")
			}
		case <-ticker.C:
			// time-based flush on comma or idle
			str := buf.String()
			if str == "" {
				continue
			}
			timeSinceAdd := time.Since(lastAdd)
			timeSinceFlush := time.Since(lastFlush)

			// bug: since idle flush sorts of disturbs the audio quality
			if timeSinceAdd >= s.IdleFlush {
				flush(true, "idle")
			} else if strings.HasSuffix(strings.TrimSpace(str), ",") && timeSinceAdd >= s.CommaDelay {
				flush(false, "comma")
			} else if timeSinceFlush >= s.ForceFlush {
				// Force periodic flush to avoid starvation when tokens flow continuously
				flush(true, "force")
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
