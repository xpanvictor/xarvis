import React, { useState, useRef, useEffect, useCallback } from 'react';
import { Send, User, Bot } from 'lucide-react';
import ReactMarkdown from 'react-markdown';

interface Message {
  id: string;
  content: string;
  role: 'user' | 'assistant';
  timestamp: Date;
  audioChunks?: Blob[];
  isStreaming?: boolean;
}

interface WebSocketMessage {
  name?: string;
  payload?: any;
  index?: number;
  text?: string;
}

export const ChatInterface: React.FC = () => {
  const [messages, setMessages] = useState<Message[]>([
    {
      id: '1',
      content: 'Hello! I\'m Xarvis, your AI assistant. I can help you with various tasks. I support real-time streaming responses with both text and audio output. How can I assist you today?',
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
  const audioChunksRef = useRef<Blob[]>([]);
  const currentAudioQueueRef = useRef<Blob[]>([]);
  const isPlayingAudioRef = useRef(false);
  const currentMessageIdRef = useRef<string>('');

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const connectWebSocket = useCallback(() => {
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
        // Handle JSON messages (text deltas and events)
        console.log("recv:", event)
        try {
          const msg: WebSocketMessage = JSON.parse(event.data);

          if (typeof msg.index === 'number' && msg.text) {
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
        console.log('Received audio frame:', event.data, 'size:', event.data.byteLength || event.data.size);
        handleAudioFrame(event.data);
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
      audioChunksRef.current = [];

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

  const handleAudioFrame = (audioData: Blob) => {
    console.log('handleAudioFrame called with:', audioData, 'current chunks count:', audioChunksRef.current.length + 1);
    audioChunksRef.current.push(audioData);

    // Add to real-time queue and play immediately
    currentAudioQueueRef.current.push(audioData);
    playNextAudioChunk();

    // Update current message with audio chunks
    setMessages(prev => prev.map(msg =>
      msg.id === currentMessageIdRef.current
        ? { ...msg, audioChunks: [...audioChunksRef.current] }
        : msg
    ));
  }; const playNextAudioChunk = async () => {
    if (isPlayingAudioRef.current || currentAudioQueueRef.current.length === 0) {
      return;
    }

    isPlayingAudioRef.current = true;
    const chunk = currentAudioQueueRef.current.shift()!;

    try {
      console.log('Playing real-time audio chunk, size:', chunk.size, 'bytes');
      const audioBlob = new Blob([chunk], { type: 'audio/wav' });
      const audioUrl = URL.createObjectURL(audioBlob);
      const audio = new Audio(audioUrl);

      await new Promise<void>((resolve, reject) => {
        audio.onended = () => {
          console.log('Real-time audio chunk finished');
          URL.revokeObjectURL(audioUrl);
          isPlayingAudioRef.current = false;
          resolve();
          // Play next chunk if available
          setTimeout(() => playNextAudioChunk(), 50);
        };

        audio.onerror = (e) => {
          console.error('Real-time audio chunk error:', e, audio.error);
          URL.revokeObjectURL(audioUrl);
          isPlayingAudioRef.current = false;
          reject(e);
        };

        audio.play().catch(reject);
      });

    } catch (error) {
      console.error('Error playing real-time audio chunk:', error);
      isPlayingAudioRef.current = false;
      // Continue with next chunk
      setTimeout(() => playNextAudioChunk(), 100);
    }
  };

  const handleEvent = (eventName: string, payload: any) => {
    console.log('Received event:', eventName, payload);

    if (eventName === 'message_complete') {
      // Mark current message as complete
      setMessages(prev => prev.map(msg =>
        msg.id === currentMessageIdRef.current
          ? { ...msg, isStreaming: false }
          : msg
      ));

      // No need to play audio here since we're playing in real-time
      console.log('Message complete, audio was played in real-time');

      setIsLoading(false);
      currentMessageRef.current = '';
      audioChunksRef.current = [];
    }
  };

  const playAudioChunks = async (chunks: Blob[]) => {
    console.log('playAudioChunks called with', chunks.length, 'chunks');

    if (chunks.length === 0) return;

    try {
      // Play each chunk sequentially since they're individual WAV files
      for (let i = 0; i < chunks.length; i++) {
        const chunk = chunks[i];
        console.log(`Playing audio chunk ${i + 1}/${chunks.length}, size: ${chunk.size} bytes`);

        const audioUrl = URL.createObjectURL(chunk);

        try {
          const audio = new Audio(audioUrl);

          // Ensure audio loads properly
          audio.preload = 'auto';

          // Wait for this chunk to finish before playing the next
          await new Promise<void>((resolve, reject) => {
            let hasResolved = false;

            const cleanup = () => {
              if (!hasResolved) {
                hasResolved = true;
                URL.revokeObjectURL(audioUrl);
              }
            };

            audio.onended = () => {
              console.log(`Audio chunk ${i + 1} finished`);
              cleanup();
              resolve();
            };

            audio.onerror = (e) => {
              console.error(`Audio chunk ${i + 1} error:`, e, audio.error);
              cleanup();
              reject(new Error(`Audio chunk ${i + 1} failed to play`));
            };

            audio.onloadstart = () => {
              console.log(`Audio chunk ${i + 1} load started`);
            };

            audio.oncanplaythrough = () => {
              console.log(`Audio chunk ${i + 1} can play through`);
            };

            // Start playing
            audio.play().catch((playError) => {
              console.error(`Audio chunk ${i + 1} play error:`, playError);
              cleanup();
              reject(playError);
            });

            // Timeout fallback in case audio gets stuck
            setTimeout(() => {
              if (!hasResolved) {
                console.warn(`Audio chunk ${i + 1} timed out`);
                cleanup();
                resolve(); // Continue to next chunk
              }
            }, 10000); // 10 second timeout per chunk
          });

        } catch (chunkError) {
          console.error(`Error with chunk ${i + 1}:`, chunkError);
          URL.revokeObjectURL(audioUrl);
          // Continue with next chunk even if this one fails
        }

        // Small delay between chunks to ensure clean transitions
        await new Promise(resolve => setTimeout(resolve, 500));
      }

      console.log('All audio chunks played successfully');
    } catch (error) {
      console.error('Error playing audio chunks:', error);
    }
  };

  const concatenateWavFiles = async (chunks: Blob[]): Promise<Blob> => {
    if (chunks.length === 0) return new Blob([], { type: 'audio/wav' });
    if (chunks.length === 1) return chunks[0];

    try {
      // Read the first chunk to get the WAV header
      const firstChunk = await chunks[0].arrayBuffer();
      const firstView = new DataView(firstChunk);

      // Verify it's a WAV file (should start with "RIFF")
      const riffHeader = String.fromCharCode(
        firstView.getUint8(0),
        firstView.getUint8(1),
        firstView.getUint8(2),
        firstView.getUint8(3)
      );

      if (riffHeader !== 'RIFF') {
        console.warn('First chunk is not a valid WAV file, falling back to simple concatenation');
        return new Blob(chunks, { type: 'audio/wav' });
      }

      // Extract header information (first 44 bytes is standard WAV header)
      const headerSize = 44;
      const header = new Uint8Array(firstChunk.slice(0, headerSize));

      // Collect all audio data (skip headers from subsequent chunks)
      const audioDataParts: Uint8Array[] = [];
      let totalDataSize = 0;

      for (let i = 0; i < chunks.length; i++) {
        const chunkBuffer = await chunks[i].arrayBuffer();

        if (i === 0) {
          // First chunk: include everything after header
          const audioData = new Uint8Array(chunkBuffer.slice(headerSize));
          audioDataParts.push(audioData);
          totalDataSize += audioData.length;
        } else {
          // Subsequent chunks: skip header (assume same 44-byte header)
          const audioData = new Uint8Array(chunkBuffer.slice(headerSize));
          audioDataParts.push(audioData);
          totalDataSize += audioData.length;
        }
      }

      // Create new WAV file with corrected size headers
      const newHeader = new Uint8Array(header);
      const newHeaderView = new DataView(newHeader.buffer);

      // Update file size (total file size - 8 bytes for RIFF header)
      newHeaderView.setUint32(4, totalDataSize + headerSize - 8, true);

      // Update data chunk size (at offset 40 for standard WAV)
      newHeaderView.setUint32(40, totalDataSize, true);

      // Combine header with all audio data
      const combinedSize = headerSize + totalDataSize;
      const combined = new Uint8Array(combinedSize);

      combined.set(newHeader, 0);
      let offset = headerSize;

      for (const part of audioDataParts) {
        combined.set(part, offset);
        offset += part.length;
      }

      return new Blob([combined], { type: 'audio/wav' });

    } catch (error) {
      console.error('Error concatenating WAV files:', error);
      // Fallback to simple concatenation
      return new Blob(chunks, { type: 'audio/wav' });
    }
  };

  const downloadAudioChunks = async (chunks: Blob[], filename: string) => {
    try {
      console.log(`Downloading ${chunks.length} audio chunks as ${filename}.wav`);

      // Properly concatenate WAV files
      const combinedBlob = await concatenateWavFiles(chunks);
      const url = URL.createObjectURL(combinedBlob);

      const a = document.createElement('a');
      a.href = url;
      a.download = `${filename}.wav`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      console.log(`Successfully downloaded combined audio file: ${filename}.wav (${combinedBlob.size} bytes)`);
    } catch (error) {
      console.error('Error downloading audio:', error);
    }
  }; useEffect(() => {
    connectWebSocket();

    return () => {
      wsRef.current?.close();
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

    // Send message via WebSocket
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
              {/* Show audio info for all assistant messages with audio chunks */}
              {message.role === 'assistant' && message.audioChunks && message.audioChunks.length > 0 && (
                <div className="message-audio">
                  <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', marginBottom: '0.25rem' }}>
                    <span style={{ fontSize: '0.75rem', color: '#888' }}>
                      Audio chunks: {message.audioChunks.length}
                      ({message.audioChunks.reduce((sum, chunk) => sum + chunk.size, 0)} bytes)
                      {message.isStreaming && ' - [STREAMING]'}
                    </span>
                  </div>
                  <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <button
                      onClick={() => playAudioChunks(message.audioChunks!)}
                      className="play-audio-button"
                    >
                      🔊 Play All Audio (Debug)
                    </button>
                    <button
                      onClick={() => downloadAudioChunks(message.audioChunks!, `xarvis-audio-${message.id}`)}
                      className="play-audio-button"
                      style={{ background: 'linear-gradient(135deg, #fbbf24 0%, #f59e0b 100%)' }}
                    >
                      📥 Download Audio
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
