// WebSocket message types matching the backend
export type MessageType =
    | 'text'
    | 'audio'
    | 'init'
    | 'listening_state'
    | 'listening_control'
    | 'response'
    | 'error'
    | 'text_delta'
    | 'audio_format'
    | 'audio_complete'
    | 'message_complete';

export interface WSMessage {
    type: MessageType;
    data?: any;
    sessionId?: string;
    sequence?: number;
    timestamp: string;
}

export interface TextMessage {
    content: string;
}

export interface AudioMessage {
    sampleRate: number;
    channels: number;
    data: ArrayBuffer;
}

export interface ListeningControl {
    action: 'start_listening' | 'stop_listening';
}

export interface ListeningStateMessage {
    mode: 'active' | 'passive';
    timestamp: string;
}

export interface ErrorMessage {
    code: string;
    message: string;
}

export interface ResponseMessage {
    content: string;
    type: 'text' | 'audio' | 'text_complete' | 'audio_complete' | 'message_complete' | 'audio_format' | 'text_delta';
    timestamp: string;
}

export interface InitMessage {
    capabilities: {
        audioSink: boolean;
        audioWrite: boolean;
        textSink: boolean;
    };
    userId?: string;
}

// WebSocket connection states
export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'error' | 'reconnecting';

// Listening states
export type ListeningState = 'idle' | 'passive' | 'active' | 'processing';

export interface WebSocketServiceEvents {
    onMessage: (message: ResponseMessage) => void;
    onListeningStateChange: (state: ListeningStateMessage) => void;
    onConnectionStateChange: (state: ConnectionState) => void;
    onError: (error: ErrorMessage) => void;
    onAudioResponse: (audioData: ArrayBuffer) => void;
    onStreamingText: (content: string, isComplete: boolean) => void;
    onStreamingAudio: (audioBlob: Blob) => void;
    onPCMAudio: (pcmData: ArrayBuffer) => void;
    onTextDelta: (index: number, text: string) => void;
    onEvent: (eventName: string, payload: any) => void;
}

class WebSocketService {
    private ws: WebSocket | null = null;
    private reconnectAttempts = 0;
    private maxReconnectAttempts = 5;
    private reconnectInterval = 1000;
    private connectionState: ConnectionState = 'disconnected';
    private userId: string | null = null;
    private sessionId: string | null = null;
    private eventHandlers: Partial<WebSocketServiceEvents> = {};

    constructor() {
        const storedUserId = localStorage.getItem('userId');
        // Validate stored userId - clear if it looks like a URL
        if (storedUserId && (storedUserId.startsWith('ws://') || storedUserId.startsWith('wss://'))) {
            console.warn('Clearing invalid userId from localStorage:', storedUserId);
            localStorage.removeItem('userId');
            this.userId = null;
        } else {
            this.userId = storedUserId;
        }
    }

    // Event handler registration
    on<K extends keyof WebSocketServiceEvents>(event: K, handler: WebSocketServiceEvents[K]) {
        this.eventHandlers[event] = handler;
    }

    off<K extends keyof WebSocketServiceEvents>(event: K) {
        delete this.eventHandlers[event];
    }

    private emit<K extends keyof WebSocketServiceEvents>(event: K, ...args: Parameters<WebSocketServiceEvents[K]>) {
        const handler = this.eventHandlers[event];
        if (handler) {
            // @ts-ignore
            handler(...args);
        }
    }

    // Connection management
    connect(userId?: string): Promise<void> {
        return new Promise(async (resolve, reject) => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                resolve();
                return;
            }

            try {
                // Get the real user ID from the authenticated user profile
                if (!userId) {
                    const { userAPI } = await import('./api');
                    const userProfile = await userAPI.getProfile();
                    userId = userProfile.id;
                    console.log('Using authenticated user ID:', userId);
                }
            } catch (error) {
                console.warn('Failed to get authenticated user profile, falling back to generated ID:', error);
            }

            this.userId = userId || this.userId || this.generateUserId();
            localStorage.setItem('userId', this.userId);

            const wsUrl = this.getWebSocketUrl();
            this.updateConnectionState('connecting');

            this.ws = new WebSocket(wsUrl);

            this.ws.onopen = (event) => {
                console.log('WebSocket connected');
                this.updateConnectionState('connected');
                this.reconnectAttempts = 0;

                // Send initialization message
                this.sendInitMessage();
                resolve();
            };

            this.ws.onmessage = (event) => {
                this.handleMessage(event);
            };

            this.ws.onclose = (event) => {
                console.log('WebSocket closed:', event.code, event.reason);
                this.updateConnectionState('disconnected');

                if (!event.wasClean && this.shouldReconnect()) {
                    this.attemptReconnect();
                }
            };

            this.ws.onerror = (event) => {
                console.error('WebSocket error:', event);
                this.updateConnectionState('error');
                reject(new Error('WebSocket connection failed'));
            };
        });
    }

    disconnect() {
        if (this.ws) {
            this.ws.close(1000, 'User disconnect');
            this.ws = null;
        }
        this.updateConnectionState('disconnected');
    }

    private getWebSocketUrl(): string {
        const baseUrl = import.meta.env.VITE_WS_URL || 'ws://localhost:8088';
        const userId = this.userId;
        const token = localStorage.getItem('accessToken');

        let url = `${baseUrl}/ws/?userId=${userId}`;
        if (token) {
            url += `&token=${encodeURIComponent(token)}`;
        }

        return url;
    }

    private generateUserId(): string {
        const userId = 'user_' + Math.random().toString(36).substr(2, 9);
        console.log('Generated new userId:', userId);
        return userId;
    }

    private sendInitMessage() {
        const initMessage: WSMessage = {
            type: 'init',
            data: {
                capabilities: {
                    audioSink: true,
                    audioWrite: true,
                    textSink: true
                },
                userId: this.userId
            },
            timestamp: new Date().toISOString()
        };

        this.send(initMessage);
    }

    private handleMessage(event: MessageEvent) {
        try {
            // Handle binary data (audio)
            if (event.data instanceof ArrayBuffer) {
                this.handleAudioData(event.data);
                return;
            }

            // Handle Blob data (audio)
            if (event.data instanceof Blob) {
                // Convert Blob to ArrayBuffer
                event.data.arrayBuffer().then(arrayBuffer => {
                    this.handleAudioData(arrayBuffer);
                });
                return;
            }

            // Handle text data - check if it's valid JSON
            let messageData: string = event.data;
            if (typeof messageData !== 'string') {
                console.warn('Received non-string, non-binary data:', typeof messageData);
                return;
            }

            // Skip empty messages
            if (!messageData.trim()) {
                return;
            }

            // Try to parse as JSON
            let message: any;
            try {
                message = JSON.parse(messageData);
                console.log('📥 Received JSON message:', message);
            } catch (parseError) {
                // If it's not JSON, treat it as streaming text content
                console.log('📥 Received non-JSON text data, treating as streaming text:', messageData);
                this.emit('onStreamingText', messageData, false);
                return;
            }

            // Handle text delta messages (like in old_ci.tsx)
            if (typeof message.index === 'number' && message.text) {
                console.log(`Text delta received: index=${message.index}, text="${message.text}"`);
                this.emit('onTextDelta', message.index, message.text);
                return;
            }

            // Handle event messages (like in old_ci.tsx)
            if (message.name && message.payload) {
                console.log(`Event received: ${message.name}`, message.payload);
                this.emit('onEvent', message.name, message.payload);
                return;
            }

            // Convert to typed WSMessage for further processing
            const wsMessage = message as WSMessage;

            // Handle parsed JSON messages
            switch (wsMessage.type) {
                case 'init':
                    // Handle init acknowledgment
                    if (wsMessage.data && wsMessage.data.sessionId) {
                        this.sessionId = wsMessage.data.sessionId;
                        console.log('Session initialized:', this.sessionId);
                    }
                    break;

                case 'audio_format':
                    // Handle audio format information and broadcast as event
                    console.log('Audio format:', wsMessage.data);
                    this.emit('onEvent', 'audio_format', wsMessage.data);
                    break;

                case 'audio_complete':
                    // Handle audio completion - signal to play collected audio
                    console.log('Audio streaming complete');
                    this.emit('onEvent', 'audio_complete', {});
                    break;

                case 'message_complete':
                    // Handle message completion - just a session indicator
                    console.log('📝 MESSAGE_COMPLETE (direct): Session complete - just an indicator');
                    this.emit('onEvent', 'message_complete', {});
                    // NO text completion here - just session indicator
                    break;

                case 'response':
                    console.log('📝 Response message received with data:', wsMessage.data);
                    const responseData = wsMessage.data as ResponseMessage;
                    console.log('📝 Response data parsed - type:', responseData.type, 'content:', responseData.content);

                    // Handle text deltas (streaming content)
                    if (responseData.type === 'text_delta') {
                        console.log('📝 TEXT_DELTA: Emitting text delta:', responseData.content);
                        this.emit('onStreamingText', responseData.content, false); // Not complete yet
                        // Don't emit onTextDelta to avoid conflicts - handleStreamingText handles this
                        break;
                    }

                    // Handle completion events that come as response messages
                    if (responseData.type === 'text_complete') {
                        console.log('📝 TEXT_COMPLETE: Text streaming complete - ONLY event that stops UI and pushes to queue');
                        this.emit('onStreamingText', '', true); // Signal completion - ONLY here!
                        this.emit('onEvent', 'text_complete', {});
                        break;
                    }

                    if (responseData.type === 'audio_complete') {
                        console.log('📝 AUDIO_COMPLETE: Audio ready for playback - should start playing');
                        this.emit('onEvent', 'audio_complete', {});
                        // NO text completion here - only audio playback
                        break;
                    }

                    if (responseData.type === 'message_complete') {
                        console.log('📝 MESSAGE_COMPLETE: Session complete - just an indicator');
                        this.emit('onEvent', 'message_complete', {});
                        // NO text completion here - just session indicator
                        break;
                    }

                    if (responseData.type === 'text') {
                        // Handle final complete text (not streaming)
                        console.log('📝 TEXT: Emitting complete text:', responseData.content);
                        this.emit('onStreamingText', responseData.content, true);
                    }

                    this.emit('onMessage', responseData);
                    break;

                case 'text':
                    // Handle complete text messages (not streaming chunks)
                    const textData = wsMessage.data as { content: string; isComplete?: boolean };
                    this.emit('onStreamingText', textData.content, textData.isComplete !== false); // Default to complete
                    break;

                case 'audio':
                    // Handle audio response metadata
                    if (wsMessage.data && wsMessage.data.url) {
                        this.handleAudioUrl(wsMessage.data.url);
                    }
                    break;

                case 'listening_state':
                    this.emit('onListeningStateChange', wsMessage.data as ListeningStateMessage);
                    break;

                case 'error':
                    this.emit('onError', wsMessage.data as ErrorMessage);
                    break;

                default:
                    console.log('Unhandled message type:', wsMessage.type, wsMessage);
            }
        } catch (error) {
            console.error('Failed to handle WebSocket message:', error, 'Raw data:', event.data);
        }
    }

    private handleAudioData(audioData: ArrayBuffer) {
        console.log('Received binary audio data:', audioData.byteLength, 'bytes');

        // Emit as PCM audio data for direct Web Audio API playback
        this.emit('onPCMAudio', audioData);

        // Also convert to Blob for fallback handling
        const audioBlob = new Blob([audioData], { type: 'audio/pcm' });
        this.emit('onStreamingAudio', audioBlob);
    }

    private handleAudioUrl(audioUrl: string) {
        // Fetch audio from URL and emit as blob
        fetch(audioUrl)
            .then(response => response.blob())
            .then(audioBlob => {
                this.emit('onStreamingAudio', audioBlob);
            })
            .catch(error => {
                console.error('Failed to fetch audio:', error);
            });
    }

    private updateConnectionState(state: ConnectionState) {
        this.connectionState = state;
        this.emit('onConnectionStateChange', state);
    }

    private shouldReconnect(): boolean {
        return this.reconnectAttempts < this.maxReconnectAttempts;
    }

    private attemptReconnect() {
        if (!this.shouldReconnect()) {
            console.log('Max reconnection attempts reached');
            return;
        }

        this.reconnectAttempts++;
        const delay = this.reconnectInterval * Math.pow(2, this.reconnectAttempts - 1); // Exponential backoff

        console.log(`Attempting to reconnect in ${delay}ms (attempt ${this.reconnectAttempts})`);
        this.updateConnectionState('reconnecting');

        setTimeout(() => {
            this.connect(this.userId || undefined);
        }, delay);
    }

    // Message sending methods
    private send(message: WSMessage) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error('WebSocket not connected');
            return;
        }

        this.ws.send(JSON.stringify(message));
    }

    sendTextMessage(content: string) {
        const message: WSMessage = {
            type: 'text',
            data: { content },
            timestamp: new Date().toISOString()
        };

        this.send(message);
    }

    sendAudioData(audioData: ArrayBuffer, sampleRate: number = 16000, channels: number = 1) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error('WebSocket not connected');
            return;
        }

        // Send binary audio data directly
        this.ws.send(audioData);
    }

    sendListeningControl(action: 'start_listening' | 'stop_listening') {
        const message: WSMessage = {
            type: 'listening_control',
            data: { action },
            timestamp: new Date().toISOString()
        };

        this.send(message);
    }

    // Getters
    getConnectionState(): ConnectionState {
        return this.connectionState;
    }

    getUserId(): string | null {
        return this.userId;
    }

    getSessionId(): string | null {
        return this.sessionId;
    }

    isConnected(): boolean {
        return this.ws?.readyState === WebSocket.OPEN;
    }
}

// Create and export singleton instance
export const webSocketService = new WebSocketService();
export default webSocketService;
