import React, { useState, useRef, useEffect, useCallback } from 'react';
import { Send, User, Bot } from 'lucide-react';
import ReactMarkdown from 'react-markdown';

interface Message {
  id: string;
  content: string;
  role: 'user' | 'assistant';
  timestamp: Date;
  audioChunks?: IndexedAudioChunk[];
  isStreaming?: boolean;
}

interface IndexedAudioChunk {
  index: number;
  data: Blob;
  received: boolean;
}

interface WebSocketMessage {
  name?: string;
  payload?: any;
  index?: number;
  text?: string;
}

interface AudioMessage {
  type: string;
  index: number;
  sessionId: string;
  size: number;
}

export const ChatInterface: React.FC = () => {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: '1',
      content: "Hello! I'm Xarvis, your AI assistant. I can help you with various tasks. I support real-time streaming responses with both text and audio output. How can I assist you today?",
      role: 'assistant',
      timestamp: new Date(),
    }
  ]);
  const [inputValue, setInputValue] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<'connecting' | 'connected' | 'disconnected'>('disconnected');
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const currentMessageRef = useRef<string>('');
  const currentMessageIdRef = useRef<string>('');

  // Simple audio state - just a queue that gets processed
  const audioContextRef = useRef<AudioContext | null>(null);
  const audioQueueRef = useRef<IndexedAudioChunk[]>([]);
  const isPlayingRef = useRef(false);
  const pendingAudioMetadataRef = useRef<AudioMessage | null>(null);

  // State to trigger useEffect when new chunks arrive
  const [audioQueue, setAudioQueue] = useState<IndexedAudioChunk[]>([]); const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Simple audio processor - pops from queue and plays
  useEffect(() => {
    const playNextChunk = async () => {
      if (isPlayingRef.current || audioQueue.length === 0) {
        return; // Already playing or no chunks to play
      }

      isPlayingRef.current = true;
      console.log('Playing next chunk from queue. Queue length:', audioQueue.length);

      // Get the first chunk from queue (don't remove it yet)
      const nextChunk = audioQueue[0];

      try {
        console.log(`Playing audio chunk ${nextChunk.index}`);

        // Use Web Audio API
        if (!audioContextRef.current) {
          audioContextRef.current = new (window.AudioContext || (window as any).webkitAudioContext)();
        }

        const arrayBuffer = await nextChunk.data.arrayBuffer();
        const audioBuffer = await audioContextRef.current.decodeAudioData(arrayBuffer);
        const source = audioContextRef.current.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(audioContextRef.current.destination);

        // Wait for chunk to finish
        await new Promise<void>((resolve) => {
          source.onended = () => {
            console.log(`Audio chunk ${nextChunk.index} finished`);
            resolve();
          };
          source.start();
        });

      } catch (error) {
        console.error(`Error playing chunk ${nextChunk.index}:`, error);
      } finally {
        isPlayingRef.current = false;
        console.log('Finished playing chunk, ready for next');

        // Remove the played chunk and trigger next play
        setAudioQueue(prev => prev.slice(1));
      }
    };

    playNextChunk();
  }, [audioQueue]); const connectWebSocket = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    setConnectionStatus('connecting');

    // Use appropriate WebSocket URL based on environment
    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsHost = window.location.host;
    const wsUrl = `${wsProtocol}//${wsHost}/ws`;

    wsRef.current = new WebSocket(wsUrl);

    wsRef.current.onopen = () => {
      console.log('WebSocket connected');
      setConnectionStatus('connected');
    };

    wsRef.current.onclose = () => {
      console.log('WebSocket disconnected');
      setConnectionStatus('disconnected');
      setIsLoading(false);
      // Attempt to reconnect after a delay
      setTimeout(connectWebSocket, 3000);
    };

    wsRef.current.onerror = (error) => {
      console.error('WebSocket error:', error);
      setConnectionStatus('disconnected');
      setIsLoading(false);
    };

    wsRef.current.onmessage = (event) => {
      if (typeof event.data === 'string') {
        // Handle JSON messages (text deltas, events, and audio metadata)
        console.log("recv JSON:", event)
        try {
          const msg = JSON.parse(event.data);

          if (msg.type === 'audio_meta') {
            // Store audio metadata for next binary message
            pendingAudioMetadataRef.current = msg as AudioMessage;
          } else if (typeof msg.index === 'number' && msg.text) {
            // Text delta message
            handleTextDelta(msg.index, msg.text);
          } else if (msg.name && msg.payload) {
            // Event message
            handleEvent(msg.name, msg.payload);
          }
        } catch (error) {
          console.error('Error parsing WebSocket message:', error);
        }
      } else {
        // Handle binary messages (audio frames)
        console.log('Received binary audio frame, size:', event.data.byteLength || event.data.size);

        if (pendingAudioMetadataRef.current) {
          // Use metadata from previous message
          handleIndexedAudioFrame(pendingAudioMetadataRef.current, event.data);
          pendingAudioMetadataRef.current = null;
        } else {
          // Fallback to legacy handling
          handleLegacyAudioFrame(event.data);
        }
      }
    };
  }, []);

  const handleTextDelta = (index: number, text: string) => {
    console.log(`handleTextDelta: index=${index}, text="${text}", currentMessageId=${currentMessageIdRef.current}`);

    if (index === 1 || !currentMessageIdRef.current) {
      // Start of new message (index starts at 1, not 0)
      const messageId = Date.now().toString();
      currentMessageIdRef.current = messageId;
      currentMessageRef.current = text;

      // Reset audio state for new message
      setAudioQueue([]);

      const newMessage: Message = {
        id: messageId,
        content: text,
        role: 'assistant',
        timestamp: new Date(),
        isStreaming: true,
        audioChunks: [],
      };

      console.log('Creating new message:', newMessage);
      setMessages(prev => {
        const newMessages = [...prev, newMessage];
        console.log('Updated messages:', newMessages);
        return newMessages;
      });
    } else {
      // Continue streaming message
      currentMessageRef.current += text;
      console.log('Updating message content to:', currentMessageRef.current);

      setMessages(prev => prev.map(msg =>
        msg.id === currentMessageIdRef.current
          ? { ...msg, content: currentMessageRef.current }
          : msg
      ));
    }
  };

  const initializeAudioContext = () => {
    if (!audioContextRef.current) {
      audioContextRef.current = new (window.AudioContext || (window as any).webkitAudioContext)();
    }
    return audioContextRef.current;
  };

  // Simplified and robust audio queue processing
  const handleIndexedAudioFrame = (audioMeta: AudioMessage, audioData: Blob) => {
    console.log('handleIndexedAudioFrame called with index:', audioMeta.index, 'data size:', audioData.size, 'expected size:', audioMeta.size);

    // Verify data integrity
    if (audioData.size !== audioMeta.size) {
      console.warn(`Audio data size mismatch: expected ${audioMeta.size}, got ${audioData.size}`);
    }

    const indexedChunk: IndexedAudioChunk = {
      index: audioMeta.index,
      data: new Blob([audioData], { type: 'audio/wav' }), // Ensure proper MIME type
      received: true
    };

    // Simply add to queue - useEffect will handle playing
    setAudioQueue(prev => {
      const newQueue = [...prev, indexedChunk].sort((a, b) => a.index - b.index);
      console.log(`Added chunk ${indexedChunk.index}. Queue now:`, newQueue.map(c => c.index));
      return newQueue;
    });

    // Update current message with accumulated audio chunks for UI
    setMessages(prev => prev.map(msg => {
      if (msg.id === currentMessageIdRef.current) {
        const existingChunks = msg.audioChunks || [];
        const updatedChunks = [...existingChunks.filter(c => c.index !== indexedChunk.index), indexedChunk];
        return { ...msg, audioChunks: updatedChunks.sort((a, b) => a.index - b.index) };
      }
      return msg;
    }));
  };

  const handleLegacyAudioFrame = (audioData: Blob) => {
    console.log('handleLegacyAudioFrame called - converting to indexed audio');

    const indexedChunk: IndexedAudioChunk = {
      index: 1, // Legacy audio is treated as single chunk
      data: new Blob([audioData], { type: 'audio/wav' }), // Ensure proper MIME type
      received: true
    };

    // Add to queue to trigger useEffect
    setAudioQueue([indexedChunk]);
    console.log('Added legacy chunk to queue');

    // Update current message with audio chunks for UI
    setMessages(prev => prev.map(msg =>
      msg.id === currentMessageIdRef.current
        ? { ...msg, audioChunks: [indexedChunk] }
        : msg
    ));
  }; const handleEvent = (eventName: string, payload: any) => {
    console.log('Received event:', eventName, payload);

    if (eventName === 'message_complete') {
      // Mark current message as complete
      setMessages(prev => prev.map(msg =>
        msg.id === currentMessageIdRef.current
          ? { ...msg, isStreaming: false }
          : msg
      ));

      console.log('Message complete. Audio queue length:', audioQueue.length);
      setIsLoading(false);
      currentMessageRef.current = '';
    } else if (eventName === 'audio_complete') {
      console.log('Audio complete event received. Total chunks:', payload?.totalChunks);
      // No special handling needed - queue will naturally finish playing
    }
  };

  // Simple function to play stored audio chunks (for UI buttons)
  const playStoredAudio = async (chunks: IndexedAudioChunk[]) => {
    console.log('Playing stored audio with', chunks.length, 'chunks');
    if (chunks.length === 0) return;

    // Set chunks to queue and let the processor handle them
    setAudioQueue(chunks.sort((a, b) => a.index - b.index));
  };

  // Simple function to download audio chunks
  const downloadAudio = (chunks: IndexedAudioChunk[], filename: string) => {
    if (chunks.length === 0) return;

    try {
      // Sort chunks by index for proper order
      const sortedChunks = [...chunks].sort((a, b) => a.index - b.index);

      if (sortedChunks.length === 1) {
        // Single chunk - download directly
        const url = URL.createObjectURL(sortedChunks[0].data);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${filename}.wav`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
      } else {
        // Multiple chunks - concatenate and download
        const combinedBlob = new Blob(
          sortedChunks.map(chunk => chunk.data),
          { type: 'audio/wav' }
        );
        const url = URL.createObjectURL(combinedBlob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${filename}-combined.wav`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
      }

      console.log(`Downloaded ${sortedChunks.length} audio chunks as ${filename}`);
    } catch (error) {
      console.error('Error downloading audio:', error);
    }
  };

  useEffect(() => {
    connectWebSocket();

    return () => {
      wsRef.current?.close();

      if (audioContextRef.current) {
        audioContextRef.current.close();
      }
    };
  }, [connectWebSocket]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim() || isLoading || connectionStatus !== 'connected') return;

    const userMessage: Message = {
      id: Date.now().toString(),
      content: inputValue,
      role: 'user',
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMessage]);
    setInputValue('');
    setIsLoading(true);

    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(inputValue);
    } else {
      setIsLoading(false);
      console.error('WebSocket is not connected');
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e as any);
    }
  };

  const adjustTextareaHeight = () => {
    if (inputRef.current) {
      inputRef.current.style.height = 'auto';
      inputRef.current.style.height = `${Math.min(inputRef.current.scrollHeight, 120)}px`;
    }
  };

  useEffect(() => {
    adjustTextareaHeight();
  }, [inputValue]);

  const getConnectionStatusColor = () => {
    switch (connectionStatus) {
      case 'connected': return '#4ade80';
      case 'connecting': return '#fbbf24';
      case 'disconnected': return '#ef4444';
      default: return '#6b7280';
    }
  };

  return (
    <div className="chat-container">
      <div className="chat-header">
        <h2 className="chat-title">Chat with Xarvis</h2>
        <p className="chat-subtitle">
          Real-time AI assistant with voice capabilities
        </p>
        <div className="connection-status">
          <div
            className="status-dot"
            style={{ backgroundColor: getConnectionStatusColor() }}
          ></div>
          <span style={{ textTransform: 'capitalize' }}>{connectionStatus}</span>
        </div>
      </div>

      <div className="messages-container">
        {messages.map((message) => (
          <div key={message.id} className={`message ${message.role}`}>
            <div className={`message-avatar ${message.role}`}>
              {message.role === 'user' ? <User size={16} /> : <Bot size={16} />}
            </div>
            <div className="message-content">
              <div className="message-text">
                {message.role === 'assistant' ? (
                  <ReactMarkdown>{message.content}</ReactMarkdown>
                ) : (
                  <p>{message.content}</p>
                )}
                {message.isStreaming && (
                  <span className="streaming-cursor">|</span>
                )}
              </div>
              {message.role === 'assistant' && message.audioChunks && message.audioChunks.length > 0 && (
                <div className="message-audio">
                  <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', marginBottom: '0.25rem' }}>
                    <span style={{ fontSize: '0.75rem', color: '#888' }}>
                      Audio chunks: {message.audioChunks.length}
                      ({message.audioChunks.reduce((sum, chunk) => sum + chunk.data.size, 0)} bytes)
                      {message.isStreaming && ' - [STREAMING]'}
                    </span>
                  </div>
                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <button
                      onClick={() => playStoredAudio(message.audioChunks!)}
                      className="play-audio-button"
                    >
                      🔊 Play All Audio (Debug)
                    </button>
                    <button
                      onClick={() => downloadAudio(message.audioChunks!, `xarvis-audio-${message.id}`)}
                      className="play-audio-button"
                      style={{ background: 'linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)' }}
                    >
                      💾 Download Audio
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        ))}

        {isLoading && (
          <div className="loading-indicator">
            <Bot size={16} />
            <span>Xarvis is responding...</span>
            <div className="loading-dots">
              <span></span>
              <span></span>
              <span></span>
            </div>
          </div>
        )}

        <div ref={messagesEndRef} />
      </div>

      <div className="input-container">
        <form onSubmit={handleSubmit} className="input-form">
          <textarea
            ref={inputRef}
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyPress={handleKeyPress}
            onInput={adjustTextareaHeight}
            placeholder={
              connectionStatus === 'connected'
                ? "Type your message to Xarvis..."
                : `Cannot send messages - ${connectionStatus}`
            }
            className="input-field"
            disabled={isLoading || connectionStatus !== 'connected'}
            rows={1}
          />
          <button
            type="submit"
            disabled={!inputValue.trim() || isLoading || connectionStatus !== 'connected'}
            className="send-button"
          >
            <Send size={18} />
          </button>
        </form>
      </div>
    </div>
  );
};
