# System Models Service

A FastAPI-based microservice that provides multiple lightweight ML models including Voice Activity Detection (VAD) using the Silero VAD model and future system-level models.

## Features

- **High Accuracy**: Uses the state-of-the-art Silero VAD model
- **Real-time Processing**: Optimized for low-latency VAD processing
- **Multiple Formats**: Supports various audio formats (WAV, MP3, etc.)
- **REST API**: Simple HTTP endpoints for easy integration
- **Docker Support**: Containerized for easy deployment
- **Health Monitoring**: Built-in health checks and monitoring

## API Endpoints

### POST /vad
Voice Activity Detection endpoint that accepts raw audio bytes.

**Parameters:**
- `file`: Audio file in bytes (required)
- `threshold`: Voice detection threshold (0.0-1.0, default: 0.5)
- `min_speech_duration_ms`: Minimum speech duration in ms (default: 250)
- `min_silence_duration_ms`: Minimum silence duration in ms (default: 100)
- `window_size_samples`: Window size for processing (default: 1536)
- `sampling_rate`: Expected sampling rate (default: 16000)

**Response:**
```json
{
  "has_voice": true,
  "confidence": 0.87,
  "segments": [
    {
      "start": 0.5,
      "end": 2.1,
      "confidence": 0.92
    }
  ],
  "processing_time_ms": 12.5,
  "audio_duration_ms": 3000.0
}
```

### POST /vad/stream
Alternative endpoint for file uploads (useful for testing).

### GET /health
Health check endpoint.

**Response:**
```json
{
  "status": "healthy",
  "model_loaded": true,
  "version": "1.0.0"
}
```

## Quick Start

### Local Development

1. **Install dependencies:**
```bash
pip install -r requirements.txt
```

2. **Run the server:**
```bash
python vad_server.py
```

The service will be available at `http://localhost:8001`

### Docker Deployment

1. **Build the image:**
```bash
docker build -t silero-vad .
```

2. **Run the container:**
```bash
docker run -p 8001:8001 silero-vad
```

### Testing

You can test the service using curl:

```bash
# Test with a WAV file
curl -X POST "http://localhost:8001/vad/stream" \
  -H "accept: application/json" \
  -H "Content-Type: multipart/form-data" \
  -F "file=@test_audio.wav"

# Test with custom parameters
curl -X POST "http://localhost:8001/vad/stream?threshold=0.3&min_speech_duration_ms=100" \
  -F "file=@test_audio.wav"
```

## Integration with Go Backend

The service is designed to integrate with the Xarvis Go backend. Here's how to call it from Go:

```go
// Example integration code
func callSileroVAD(audioData []byte, config VADConfig) (VADResult, error) {
    url := "http://silero-vad:8001/vad"
    
    // Create form data
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    
    // Add file
    part, err := writer.CreateFormFile("file", "audio.wav")
    if err != nil {
        return VADResult{}, err
    }
    part.Write(audioData)
    
    // Add parameters
    writer.WriteField("threshold", fmt.Sprintf("%.2f", config.Threshold))
    writer.WriteField("sampling_rate", fmt.Sprintf("%d", config.SampleRate))
    
    writer.Close()
    
    // Make request
    req, err := http.NewRequest("POST", url, body)
    if err != nil {
        return VADResult{}, err
    }
    req.Header.Set("Content-Type", writer.FormDataContentType())
    
    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return VADResult{}, err
    }
    defer resp.Body.Close()
    
    // Parse response
    var vadResponse VADResponse
    if err := json.NewDecoder(resp.Body).Decode(&vadResponse); err != nil {
        return VADResult{}, err
    }
    
    return VADResult{
        HasVoice:   vadResponse.HasVoice,
        Confidence: vadResponse.Confidence,
    }, nil
}
```

## Configuration

The service can be configured through environment variables:

- `HOST`: Server host (default: "0.0.0.0")
- `PORT`: Server port (default: 8001)
- `LOG_LEVEL`: Logging level (default: "info")

## Performance

- **Model Loading**: ~2-3 seconds on first startup
- **Processing Time**: ~10-50ms for 1-3 second audio clips
- **Memory Usage**: ~200-400MB including model
- **Throughput**: Handles multiple concurrent requests efficiently

## Architecture Notes

- Uses PyTorch and the official Silero VAD model
- Optimized for 16kHz audio (recommended)
- Supports batch processing for efficiency
- Built-in audio format conversion using torchaudio
- Graceful error handling and logging
- Ready for production deployment with proper health checks
