package stream

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
)

func ConvertAudioToMP3(wav io.Reader, ct string) (io.Reader, error) {
	if ct == "audio/mp3" {
		return wav, nil
	}

	log.Printf("Output type: %s", ct)

	// Read all WAV data - this is now safe since we pass a bytes.Reader
	wavBytes, err := io.ReadAll(wav)
	if err != nil {
		return nil, fmt.Errorf("failed to read WAV: %w", err)
	}

	log.Printf("Read %d WAV bytes, converting to MP3", len(wavBytes))

	if len(wavBytes) == 0 {
		return nil, fmt.Errorf("received empty WAV data")
	}

	// extract format from ct
	ioFmt := strings.Split(ct, "/")[1]
	log.Printf("format: %v", ioFmt)

	cmd := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "error",
		"-f", "wav", // <-- explicitly tell ffmpeg what format is coming
		"-i", "pipe:0",
		"-f", "mp3",
		"pipe:1",
	)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(wavBytes)
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		log.Printf("ffmpeg stderr: %s", stderr.String())
		return nil, fmt.Errorf("conversion to mp3 error: %w", err)
	}

	log.Printf("Successfully converted to MP3: %d bytes", out.Len())
	return &out, nil
}
