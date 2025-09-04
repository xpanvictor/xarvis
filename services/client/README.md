# Xarvis Web Client

A modern React-based web interface for Xarvis, built with Vite and TypeScript.

## Features

- **Chat Interface**: Clean, modern chat UI with support for text input and audio output
- **Real-time Communication**: Direct integration with Xarvis API endpoints
- **Audio Playback**: Automatic audio playback for assistant responses
- **Responsive Design**: Works on desktop and mobile devices
- **Markdown Support**: Rich text rendering for assistant responses
- **Dark Theme**: Sleek dark theme with glassmorphism effects

## Development

### Prerequisites

- Node.js 18+
- npm or yarn

### Local Development

```bash
# Install dependencies
npm install

# Start development server
npm run dev
```

The app will be available at `http://localhost:5173`

### Docker Development

```bash
# From the project root, run with client profile
docker-compose --profile client up
```

### Production Build

```bash
# Build for production
npm run build

# Preview production build
npm run preview
```

### Docker Production

```bash
# From the project root, run with client-prod profile
docker-compose --profile client-prod up
```

## API Integration

The client connects to Xarvis API endpoints:

- **Chat**: `POST /v1/app/chat` - Send messages and receive responses
- **Audio**: Audio URLs returned in chat responses for TTS playback

### Chat API Format

```json
{
  "message": "User input text",
  "conversation_id": "web-client-session"
}
```

Response:
```json
{
  "message": "Assistant response text",
  "audio_url": "http://example.com/audio.mp3"
}
```

## Architecture

- **React 18**: Modern React with hooks
- **TypeScript**: Type-safe development
- **Vite**: Fast build tool and dev server
- **Lucide React**: Modern icon library
- **React Markdown**: Markdown rendering for rich responses

## Deployment

The client can be deployed as:

1. **Development**: Vite dev server with hot reload
2. **Production**: Static build served by Nginx with API proxying

## Configuration

- **Vite Config**: API proxy settings for development
- **Nginx Config**: Production server with API routing
- **Docker**: Multi-stage builds for dev and prod environments
