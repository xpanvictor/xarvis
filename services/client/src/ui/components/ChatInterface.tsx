import React, { useState, useRef, useEffect, useCallback } from 'react';
import { Send, User, Bot } from 'lucide-react';
import ReactMarkdown from 'react-markdown';

interface Message {
  id: string;
  content: string;
  role: 'user' | 'assistant';
  timestamp: Date;
  audioChunks?: AudioChunk[];
  isStreaming?: boolean;
}

interface AudioChunk {
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
  const pendingAudioMetadataRef = useRef<AudioMessage | null>(null);

  // Audio queue and real-time playback
  const [audioQueue, setAudioQueue] = useState<AudioChunk[]>([]);
  const audioQueueRef = useRef<AudioChunk[]>([]);
  const audioContextRef = useRef<AudioContext | null>(null);
  const isPlayingRef = useRef<boolean>(false);

  // Audio buffering for MP3 streams
  const audioBufferRef = useRef<{
    chunks: Blob[];
    headerChunk: Blob | null;
    currentIndex: number;
  }>({
    chunks: [],
    headerChunk: null,
    currentIndex: 0
  });

  // Keep ref in sync with state
  useEffect(() => {
    audioQueueRef.current = audioQueue;
  }, [audioQueue]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  // Real-time audio player - automatically plays chunks from queue
  const playAudioQueue = async () => {
    // If already playing, ignore trigger
    if (isPlayingRef.current) {
      return;
    }

    // Check if we have chunks to play using ref
    if (audioQueueRef.current.length === 0) {
      return;
    }

    isPlayingRef.current = true;
    console.log(`🎵 Starting audio playback. Queue length: ${audioQueueRef.current.length}`);

    try {
      // Initialize audio context if needed
      if (!audioContextRef.current) {
        audioContextRef.current = new (window.AudioContext || (window as any).webkitAudioContext)();
      }

      // Resume audio context if suspended (browser policy)
      if (audioContextRef.current.state === 'suspended') {
        await audioContextRef.current.resume();
      }

      // Start playing the first chunk
      playNextChunkRecursive();

    } catch (error) {
      console.error('❌ Audio playback error:', error);
      isPlayingRef.current = false;
    }
  };

  // Recursive function to play chunks one by one
  const playNextChunkRecursive = () => {
    // Check if we should continue playing
    if (!isPlayingRef.current) {
      console.log('🛑 Playback stopped');
      return;
    }

    // Get current queue from ref (always current)
    const currentQueue = audioQueueRef.current;

    // If queue is empty, stop playback
    if (currentQueue.length === 0) {
      console.log(`🎉 Audio playback complete. Queue empty.`);
      isPlayingRef.current = false;
      return;
    }

    // Get the first chunk
    const currentChunk = currentQueue[0];
    console.log(`🔊 Playing chunk ${currentChunk.index} (${(currentChunk.data.size / 1024).toFixed(2)} KB) - Queue: [${currentQueue.map(c => c.index).join(', ')}]`);

    // Play the chunk and handle completion
    playChunkAudio(currentChunk)
      .then(() => {
        console.log(`✅ Chunk ${currentChunk.index} finished playing`);

        // Remove played chunk from queue
        setAudioQueue(prev => {
          const newQueue = prev.slice(1);
          console.log(`🗑️ Removed chunk ${currentChunk.index}. Remaining: ${newQueue.length} chunks`);
          return newQueue;
        });

        // Continue with next chunk after brief delay
        setTimeout(() => playNextChunkRecursive(), 50);
      })
      .catch(error => {
        console.error(`❌ Failed to play chunk ${currentChunk.index}:`, error);

        // Remove failed chunk from queue
        setAudioQueue(prev => {
          const newQueue = prev.slice(1);
          console.log(`🗑️ Removed failed chunk ${currentChunk.index}. Remaining: ${newQueue.length} chunks`);
          return newQueue;
        });

        // Continue with next chunk after brief delay
        setTimeout(() => playNextChunkRecursive(), 50);
      });
  };

  // Actually play the audio data with better error handling
  const playChunkAudio = async (chunk: AudioChunk): Promise<void> => {
    if (!audioContextRef.current) {
      throw new Error('Audio context not available');
    }

    try {
      const arrayBuffer = await chunk.data.arrayBuffer();

      // Check if we have valid audio data
      if (arrayBuffer.byteLength === 0) {
        throw new Error('Empty audio buffer');
      }

      const audioBuffer = await audioContextRef.current.decodeAudioData(arrayBuffer);

      const source = audioContextRef.current.createBufferSource();
      source.buffer = audioBuffer;
      source.connect(audioContextRef.current.destination);

      return new Promise<void>((resolve, reject) => {
        const timeout = setTimeout(() => {
          console.error(`⏰ Chunk ${chunk.index} playback timeout`);
          reject(new Error('Playback timeout'));
        }, 30000);

        source.onended = () => {
          clearTimeout(timeout);
          resolve();
        };

        source.start();
      });

    } catch (error) {
      // Enhanced error logging for audio decode issues
      if (error instanceof DOMException && error.message.includes('unknown content type')) {
        console.error(`🔍 Chunk ${chunk.index} decode error - likely corrupted or invalid audio data`);
        console.error(`🔍 Chunk size: ${chunk.data.size} bytes, MIME type: ${chunk.data.type}`);
      }
      throw error;
    }
  };  // Trigger audio playback when queue changes (new chunks arrive)
  useEffect(() => {
    playAudioQueue();
  }, [audioQueue]);

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
        // Handle binary messages (audio frames) - store only, no playback
        console.log('Received binary audio frame, size:', event.data.byteLength || event.data.size);

        if (pendingAudioMetadataRef.current) {
          // Use metadata from previous message
          handleAudioFrame(pendingAudioMetadataRef.current, event.data);
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

      // Reset audio queue for new message
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

  // Audio format detection utility - optimized for known MP3 streams
  const detectAudioFormat = async (blob: Blob, chunkIndex: number): Promise<string> => {
    // Since backend always sends MP3, we can optimize this
    if (blob.size < 12) {
      console.log(`Chunk ${chunkIndex}: Too small for header analysis, using audio/mpeg`);
      return 'audio/mpeg';
    }

    const arrayBuffer = await blob.slice(0, 12).arrayBuffer();
    const bytes = new Uint8Array(arrayBuffer);

    // Quick check for MP3 headers (ID3 tag or MP3 frame sync)
    if (bytes[0] === 0x49 && bytes[1] === 0x44 && bytes[2] === 0x33) {
      console.log(`Chunk ${chunkIndex}: MP3 with ID3 header detected`);
      return 'audio/mpeg';
    }
    if (bytes[0] === 0xFF && (bytes[1] & 0xE0) === 0xE0) {
      console.log(`Chunk ${chunkIndex}: MP3 frame sync detected`);
      return 'audio/mpeg';
    }

    // For known MP3 stream, log but don't warn
    console.log(`Chunk ${chunkIndex}: MP3 streaming data (no header), size: ${blob.size} bytes`);
    return 'audio/mpeg';
  };

  // Check if an audio chunk is individually playable (has complete headers)
  const isChunkPlayable = async (blob: Blob): Promise<boolean> => {
    if (blob.size < 12) return false;

    const arrayBuffer = await blob.slice(0, 12).arrayBuffer();
    const bytes = new Uint8Array(arrayBuffer);

    // Check for MP3 ID3 header or MP3 frame sync at the beginning
    const hasID3Header = bytes[0] === 0x49 && bytes[1] === 0x44 && bytes[2] === 0x33;
    const hasMP3Sync = bytes[0] === 0xFF && (bytes[1] & 0xE0) === 0xE0;

    return hasID3Header || hasMP3Sync;
  };  // Handle audio frames - buffer chunks and create complete audio files
  const handleAudioFrame = async (audioMeta: AudioMessage, audioData: Blob) => {
    const chunkSizeKB = (audioData.size / 1024).toFixed(2);
    console.log(`📦 Audio chunk ${audioMeta.index}: ${chunkSizeKB} KB`);

    // Verify data integrity
    if (audioData.size !== audioMeta.size) {
      console.warn(`⚠️ Size mismatch chunk ${audioMeta.index}: expected ${audioMeta.size}, got ${audioData.size}`);
    }

    // Skip empty or very small chunks that are likely corrupted
    if (audioData.size < 100) {
      console.warn(`⚠️ Chunk ${audioMeta.index} too small (${audioData.size} bytes), skipping`);
      return;
    }

    try {
      const isPlayable = await isChunkPlayable(audioData);

      if (isPlayable) {
        // This chunk has a complete header - process any buffered chunks first
        await flushAudioBuffer();

        // Store this as the new header chunk for future streaming data
        audioBufferRef.current.headerChunk = audioData;
        audioBufferRef.current.currentIndex = audioMeta.index;

        // Add this complete chunk to the queue
        const audioChunk: AudioChunk = {
          index: audioMeta.index,
          data: new Blob([audioData], { type: 'audio/mpeg' }),
          received: true
        };

        setAudioQueue(prev => {
          const newQueue = [...prev, audioChunk].sort((a, b) => a.index - b.index);
          console.log(`🎵 Queue updated: ${newQueue.length} chunks [${newQueue.map(c => c.index).join(', ')}]`);
          return newQueue;
        });

        console.log(`✅ Added complete chunk ${audioMeta.index} to queue`);
      } else {
        // This is streaming data - buffer it
        audioBufferRef.current.chunks.push(audioData);
        console.log(`🔄 Buffering chunk ${audioMeta.index} (${audioBufferRef.current.chunks.length} buffered chunks)`);

        // If we have enough buffered chunks or this is the last expected chunk, flush the buffer
        if (audioBufferRef.current.chunks.length >= 5) {
          await flushAudioBuffer();
        }
      }

      // Trigger playback if not already playing
      if (!isPlayingRef.current) {
        setTimeout(() => playAudioQueue(), 100);
      }

    } catch (error) {
      console.error(`Error processing audio chunk ${audioMeta.index}:`, error);
    }
  };

  // Flush buffered audio chunks by combining them with the last header
  const flushAudioBuffer = async () => {
    const buffer = audioBufferRef.current;

    if (buffer.chunks.length === 0) {
      return; // Nothing to flush
    }

    if (!buffer.headerChunk) {
      console.warn('⚠️ No header chunk available for buffered audio data, skipping buffer flush');
      buffer.chunks = []; // Clear buffer
      return;
    }

    try {
      // Extract header from the last complete chunk
      const headerData = await extractMP3Header(buffer.headerChunk);

      if (!headerData) {
        console.warn('⚠️ Could not extract header from reference chunk');
        buffer.chunks = []; // Clear buffer
        return;
      }

      // Combine header with all buffered chunks
      const combinedChunks = [headerData, ...buffer.chunks];
      const combinedBlob = new Blob(combinedChunks, { type: 'audio/mpeg' });

      // Create a new audio chunk with combined data
      const combinedIndex = buffer.currentIndex + buffer.chunks.length;
      const audioChunk: AudioChunk = {
        index: combinedIndex,
        data: combinedBlob,
        received: true
      };

      setAudioQueue(prev => {
        const newQueue = [...prev, audioChunk].sort((a, b) => a.index - b.index);
        console.log(`🎵 Queue updated with buffered audio: ${newQueue.length} chunks [${newQueue.map(c => c.index).join(', ')}]`);
        return newQueue;
      });

      console.log(`✅ Flushed ${buffer.chunks.length} buffered chunks as combined audio chunk ${combinedIndex}`);

      // Clear the buffer
      buffer.chunks = [];
      buffer.currentIndex = combinedIndex;

    } catch (error) {
      console.error('Error flushing audio buffer:', error);
      buffer.chunks = []; // Clear buffer on error
    }
  };

  // Extract MP3 header from a complete audio chunk
  const extractMP3Header = async (blob: Blob): Promise<Blob | null> => {
    try {
      const arrayBuffer = await blob.arrayBuffer();
      const bytes = new Uint8Array(arrayBuffer);

      // Look for ID3 header
      if (bytes[0] === 0x49 && bytes[1] === 0x44 && bytes[2] === 0x33) {
        // ID3v2 header found
        const headerSize = ((bytes[6] & 0x7F) << 21) | ((bytes[7] & 0x7F) << 14) |
          ((bytes[8] & 0x7F) << 7) | (bytes[9] & 0x7F) + 10;

        // Include ID3 header + first MP3 frame for proper header
        const headerEndIndex = Math.min(headerSize + 1024, arrayBuffer.byteLength);
        return new Blob([arrayBuffer.slice(0, headerEndIndex)], { type: 'audio/mpeg' });
      }

      // Look for MP3 frame sync and extract first frame as header
      for (let i = 0; i < Math.min(1024, bytes.length - 1); i++) {
        if (bytes[i] === 0xFF && (bytes[i + 1] & 0xE0) === 0xE0) {
          // Found MP3 frame sync, extract first 1KB as header
          const headerEndIndex = Math.min(i + 1024, arrayBuffer.byteLength);
          return new Blob([arrayBuffer.slice(0, headerEndIndex)], { type: 'audio/mpeg' });
        }
      }

      return null;
    } catch (error) {
      console.error('Error extracting MP3 header:', error);
      return null;
    }
  };

  // Handle legacy audio frames
  const handleLegacyAudioFrame = async (audioData: Blob) => {
    console.log('handleLegacyAudioFrame called - converting to indexed audio');

    try {
      const mimeType = await detectAudioFormat(audioData, 1);
      console.log(`Detected MIME type for legacy chunk:`, mimeType);

      const audioChunk: AudioChunk = {
        index: 1, // Legacy audio is treated as single chunk
        data: new Blob([audioData], { type: mimeType }),
        received: true
      };

      // Add to queue and log length
      setAudioQueue([audioChunk]);
      console.log(`🎵 Queue updated (legacy): 1 chunk [1]`);

      // Update current message with audio indicator for UI
      setMessages(prev => prev.map(msg =>
        msg.id === currentMessageIdRef.current
          ? { ...msg, audioChunks: [{ index: 1, data: new Blob(), received: true }] }
          : msg
      ));

    } catch (error) {
      console.error('Error processing legacy audio frame:', error);
    }
  };

  const handleEvent = (eventName: string, payload: any) => {
    console.log('Received event:', eventName, payload);

    if (eventName === 'message_complete') {
      // Flush any remaining buffered chunks before marking message complete
      flushAudioBuffer();

      // Mark current message as complete
      setMessages(prev => prev.map(msg =>
        msg.id === currentMessageIdRef.current
          ? { ...msg, isStreaming: false }
          : msg
      ));

      console.log(`Message complete. Audio queue length: ${audioQueue.length}`);
      setIsLoading(false);
      currentMessageRef.current = '';
    } else if (eventName === 'audio_complete') {
      console.log(`Audio complete event received. Total chunks: ${payload?.totalChunks}. Current queue length: ${audioQueue.length}`);
    }
  };

  useEffect(() => {
    connectWebSocket();

    return () => {
      wsRef.current?.close();

      // Clean up audio context
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
          Real-time AI assistant with streaming text and audio
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
                      🎵 Audio playing in real-time ({message.audioChunks.length} chunks processed)
                      {message.isStreaming && ' - [STREAMING]'}
                    </span>
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
