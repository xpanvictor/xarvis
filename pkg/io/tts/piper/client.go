package piper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Piper struct {
	BaseURL  string        // e.g. "http://tts:5000"
	Client   *http.Client  // inject; default if nil
	Voice    string        // default voice (override per-call)
	Format   string        // "wav" or "pcm_s16le"
	Rate     int           // 16000
	Channels int           // 1
	Timeout  time.Duration // request timeout per chunk
}

func New(bu string) Piper {
	return Piper{BaseURL: bu}
}

type ttsReq struct {
	Text      string      `json:"text"`
	Voice     string      `json:"voice,omitempty"`
	SpeakerID *int        `json:"speaker_id,omitempty"`
	Audio     interface{} `json:"audio,omitempty"` // map[string]any{"format":"wav","rate":16000,"channels":1}
	// Optional: SSML if your server supports it
	// Ssml bool `json:"ssml,omitempty"`
}

func (p *Piper) DoTTS(ctx context.Context, text string, optVoice string) (io.ReadCloser, string, error) {
	if text == "" {
		return nil, "", fmt.Errorf("empty text")
	}
	voice := p.Voice
	if optVoice != "" {
		voice = optVoice
	}

	body, _ := json.Marshal(ttsReq{
		Text:  text,
		Voice: voice,
		Audio: map[string]any{
			"format":   ifEmpty(p.Format, "wav"),
			"rate":     ifZero(p.Rate, 16000),
			"channels": ifZero(p.Channels, 1),
		},
	})

	req, err := http.NewRequestWithContext(ctx, "POST", p.BaseURL+"/api/tts", bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")

	hc := p.Client
	if hc == nil {
		hc = &http.Client{}
	}

	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Context timeout per chunk
	ctx2, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req = req.WithContext(ctx2)

	resp, err := hc.Do(req)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, "", fmt.Errorf("tts http %d: %s", resp.StatusCode, string(b))
	}
	// caller must Close the body
	ct := resp.Header.Get("Content-Type") // e.g. audio/wav
	return resp.Body, ct, nil
}

func ifEmpty(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
func ifZero(n, d int) int {
	if n == 0 {
		return d
	}
	return n
}
