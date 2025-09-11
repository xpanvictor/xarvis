# Refactored Audio/Text WebSocket Routes

## Overview
The routes have been refactored to properly support both audio and text input with separate endpoint management and proper capability matching.

## Key Changes

### 1. **Message Structure**
```go
type WSMessage struct {
    Type      MessageType `json:"type"`     // "text", "audio", "init"
    Data      interface{} `json:"data"`     // Message payload
    SessionID string      `json:"sessionId"`
    Sequence  int         `json:"sequence"`
}
```

### 2. **Connection Management**
- **UserConnection**: Tracks per-user state including brain system, audio buffer, and endpoints
- **Separate Endpoints**: Text and audio endpoints with specific capabilities
- **Connection Registry**: Proper cleanup and connection tracking

### 3. **Audio Input Processing**
- **PCM Frame Format**: First 8 bytes contain metadata (sample rate, channels)
- **Audio Ring Buffer**: Stores audio for future voice system processing
- **Binary Message Protocol**: Efficient audio data transmission

### 4. **Three WebSocket Endpoints**

#### `/ws` - Full-featured endpoint
- Supports both text and audio
- Creates separate text/audio endpoints
- Full brain system integration

#### `/ws/audio` - Audio-only endpoint  
- AudioSink + AudioWrite capabilities
- Optimized for audio streaming
- 2MB audio buffer

#### `/ws/text` - Text-only endpoint
- TextSink capability only
- Lightweight for text-only clients
- Direct brain system processing

## Usage Examples

### JavaScript Client - Full Endpoint
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

// Send text message
ws.send(JSON.stringify({
    type: 'text',
    data: { content: 'Hello, how are you?' }
}));

// Send audio (PCM frames)
const audioData = new Uint8Array(pcmFrames.length + 8);
const view = new DataView(audioData.buffer);
view.setUint32(0, sampleRate, true);  // 44100
view.setUint16(4, channels, true);    // 1 or 2
audioData.set(pcmFrames, 8);
ws.send(audioData);
```

### Audio-Only Client
```javascript
const audioWs = new WebSocket('ws://localhost:8080/ws/audio');

// Send only PCM audio frames
const sendAudio = (pcmData, sampleRate = 16000, channels = 1) => {
    const audioMsg = new Uint8Array(pcmData.length + 8);
    const view = new DataView(audioMsg.buffer);
    view.setUint32(0, sampleRate, true);
    view.setUint16(4, channels, true);
    audioMsg.set(pcmData, 8);
    audioWs.send(audioMsg);
};
```

### Text-Only Client
```javascript
const textWs = new WebSocket('ws://localhost:8080/ws/text');

// Send plain text messages
textWs.send('What is the weather like today?');
```

## Capability Management

The registry now properly handles endpoint selection based on capabilities:

- **Audio Input**: Uses endpoints with `AudioWrite: true`
- **Audio Output**: Uses endpoints with `AudioSink: true`  
- **Text Output**: Uses endpoints with `TextSink: true`

## Future Integration Points

1. **Voice Streaming System**: Audio buffer ready for VAD/STT integration
2. **Multiple Device Support**: MRU endpoint selection works across devices
3. **Session Management**: Proper session tracking for conversation continuity
4. **Error Handling**: Comprehensive error reporting and connection cleanup

## Benefits

1. **Separation of Concerns**: Audio and text processing are properly separated
2. **Scalable Architecture**: Each user gets isolated connection and brain system
3. **Efficient Memory Usage**: Ring buffer prevents audio memory growth
4. **Flexible Client Support**: Clients can choose appropriate endpoint
5. **Future-Ready**: Structure supports voice system integration
