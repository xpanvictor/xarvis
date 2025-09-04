package piper

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "net/url"
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

func (p *Piper) DoTTS(ctx context.Context, text string, optVoice string) (io.ReadCloser, string, error) {
	if text == "" {
		return nil, "", fmt.Errorf("empty text")
	}
	voice := p.Voice
	if optVoice != "" {
		voice = optVoice
	}

    // rhasspy/wyoming-piper HTTP: GET /api/text-to-speech?text=...&voice=...
    // Note: This API streams a WAV body on success.
    u, err := url.Parse(p.BaseURL + "/api/text-to-speech")
    if err != nil {
        return nil, "", err
    }
    q := u.Query()
    q.Set("text", text)
    if voice != "" {
        q.Set("voice", voice)
    }
    u.RawQuery = q.Encode()

    req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
    if err != nil {
        return nil, "", err
    }
    req.Header.Set("Accept", "audio/wav")

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

	start := time.Now()
    resp, err := hc.Do(req)
    if err != nil {
        return nil, "", fmt.Errorf("tts http request failed: %w (url=%s)", err, u.String())
    }
    if resp.StatusCode != http.StatusOK {
        b, _ := io.ReadAll(resp.Body)
        resp.Body.Close()
        return nil, "", fmt.Errorf("tts http %d: %s (url=%s, dur=%s)", resp.StatusCode, string(b), u.String(), time.Since(start))
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
