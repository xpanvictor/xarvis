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

  // PCM audio format info (received from backend)
  const audioFormatRef = useRef<{
    format: string;
    sampleRate: number;
    channels: number;
    bitsPerSample: number;
    encoding: string;
  } | null>(null);

  // Continuous audio streaming state
  const audioStreamRef = useRef<{
    currentTime: number;
    nextStartTime: number;
    isStreaming: boolean;
  }>({
    currentTime: 0,
    nextStartTime: 0,
    isStreaming: false
  });

  // Track active audio sources for cleanup
  const activeAudioSourcesRef = useRef<AudioBufferSourceNode[]>([]);

  // Cleanup function to stop all active audio sources
  const stopAllAudio = () => {
    console.log(`🛑 Stopping ${activeAudioSourcesRef.current.length} active audio sources`);
    activeAudioSourcesRef.current.forEach(source => {
      try {
        source.stop();
      } catch (e) {
        // Source might already be stopped
      }
    });
    activeAudioSourcesRef.current = [];
    isPlayingRef.current = false;
    audioStreamRef.current.isStreaming = false;
  };  // Keep ref in sync with state
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
    if (isPlayingRef.current || audioStreamRef.current.isStreaming) {
      console.log(`🚫 Playback already active - ignoring trigger`);
      return;
    }

    // Safety cleanup - stop any lingering audio sources
    stopAllAudio();

    // Wait for a minimum buffer of chunks to reduce stuttering
    const minBufferChunks = 2;
    if (audioQueueRef.current.length < minBufferChunks) {
      console.log(`🔄 Waiting for buffer... (${audioQueueRef.current.length}/${minBufferChunks} chunks)`);
      return;
    }

    isPlayingRef.current = true;
    audioStreamRef.current.isStreaming = true;
    console.log(`🎵 Starting continuous audio playback. Queue length: ${audioQueueRef.current.length}`);

    try {
      // Initialize audio context if needed
      if (!audioContextRef.current) {
        audioContextRef.current = new (window.AudioContext || (window as any).webkitAudioContext)();
      }

      // Resume audio context if suspended (browser policy)
      if (audioContextRef.current.state === 'suspended') {
        await audioContextRef.current.resume();
      }

      // Initialize timing for continuous playback
      audioStreamRef.current.currentTime = audioContextRef.current.currentTime;
      audioStreamRef.current.nextStartTime = audioStreamRef.current.currentTime + 0.1; // Start slightly in future

      // Start continuous chunk scheduling
      scheduleNextChunk();

    } catch (error) {
      console.error('❌ Audio playback error:', error);
      isPlayingRef.current = false;
      audioStreamRef.current.isStreaming = false;
    }
  };

  // Schedule chunks for continuous playback without gaps
  const scheduleNextChunk = () => {
    if (!audioStreamRef.current.isStreaming || !audioContextRef.current) {
      return;
    }

    // Get current queue
    const currentQueue = audioQueueRef.current;

    // If queue is empty, stop playback
    if (currentQueue.length === 0) {
      console.log(`🎉 Continuous audio playback complete. Queue empty.`);
      isPlayingRef.current = false;
      audioStreamRef.current.isStreaming = false;
      return;
    }

    // Get the first chunk
    const currentChunk = currentQueue[0];
    console.log(`🔊 Scheduling chunk ${currentChunk.index} at time ${audioStreamRef.current.nextStartTime.toFixed(3)}`);

    // Schedule this chunk for continuous playback
    playChunkContinuous(currentChunk, audioStreamRef.current.nextStartTime)
      .then((duration) => {
        console.log(`✅ Chunk ${currentChunk.index} scheduled successfully (${duration.toFixed(3)}s)`);

        // Update next start time for seamless continuity
        audioStreamRef.current.nextStartTime += duration;

        // Remove played chunk from queue
        setAudioQueue(prev => {
          const newQueue = prev.slice(1);
          console.log(`🗑️ Removed chunk ${currentChunk.index}. Remaining: ${newQueue.length} chunks`);
          return newQueue;
        });

        // Schedule next chunk immediately (no delay!)
        setTimeout(() => scheduleNextChunk(), 10);
      })
      .catch(error => {
        console.error(`❌ Failed to schedule chunk ${currentChunk.index}:`, error);

        // Remove failed chunk and continue
        setAudioQueue(prev => {
          const newQueue = prev.slice(1);
          console.log(`🗑️ Removed failed chunk ${currentChunk.index}. Remaining: ${newQueue.length} chunks`);
          return newQueue;
        });

        // Continue with next chunk
        setTimeout(() => scheduleNextChunk(), 10);
      });
  };

  // Play PCM audio chunk with precise timing for continuous playback
  const playChunkContinuous = async (chunk: AudioChunk, startTime: number): Promise<number> => {
    if (!audioContextRef.current) {
      throw new Error('Audio context not available');
    }

    if (!audioFormatRef.current) {
      throw new Error('Audio format not received yet');
    }

    try {
      const arrayBuffer = await chunk.data.arrayBuffer();

      // Check if we have valid audio data
      if (arrayBuffer.byteLength === 0) {
        throw new Error('Empty audio buffer');
      }

      const format = audioFormatRef.current;
      const samplesCount = arrayBuffer.byteLength / (format.bitsPerSample / 8);

      // Create AudioBuffer for PCM data
      const audioBuffer = audioContextRef.current.createBuffer(
        format.channels,
        samplesCount,
        format.sampleRate
      );

      // Convert raw PCM bytes to float32 samples
      const channelData = audioBuffer.getChannelData(0);

      if (format.bitsPerSample === 16 && format.encoding === 's16le') {
        // 16-bit signed little-endian PCM
        const samples = new Int16Array(arrayBuffer);
        for (let i = 0; i < samples.length; i++) {
          channelData[i] = samples[i] / 32768; // Convert to [-1, 1] range
        }
      } else {
        throw new Error(`Unsupported PCM format: ${format.bitsPerSample}-bit ${format.encoding}`);
      }

      // Schedule the audio buffer to play at precise time
      const source = audioContextRef.current.createBufferSource();
      source.buffer = audioBuffer;
      source.connect(audioContextRef.current.destination);

      // Track this source for cleanup
      activeAudioSourcesRef.current.push(source);

      // Remove from tracking when it ends
      source.onended = () => {
        const index = activeAudioSourcesRef.current.indexOf(source);
        if (index > -1) {
          activeAudioSourcesRef.current.splice(index, 1);
        }
      };

      // Start at the precise scheduled time for seamless playback
      source.start(startTime);

      // Return the duration of this chunk for timing next chunk
      const duration = audioBuffer.duration;
      console.log(`⏰ Chunk ${chunk.index} duration: ${duration.toFixed(3)}s, scheduled at: ${startTime.toFixed(3)}s`);

      return duration;

    } catch (error: any) {
      console.error(`🔍 PCM Chunk ${chunk.index} scheduling error:`, error);
      console.error(`🔍 Chunk size: ${chunk.data.size} bytes`);
      throw error;
    }
  };  // Trigger audio playback intelligently - only when we have buffer AND not already playing
  useEffect(() => {
    // Only trigger if:
    // 1. Not already playing/streaming
    // 2. Have enough chunks in buffer 
    // 3. Actually have chunks to play
    if (!isPlayingRef.current && 
        !audioStreamRef.current.isStreaming && 
        audioQueue.length >= 2) {
      console.log(`🚀 Triggering playback: ${audioQueue.length} chunks in buffer`);
      playAudioQueue();
    }
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
      // Start of new message (index starts at 1, not 0) - stop any existing audio
      console.log('🔄 New message starting - stopping previous audio');
      stopAllAudio();
      
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

  // Handle audio frames - simple PCM chunk queuing
  const handleAudioFrame = async (audioMeta: AudioMessage, audioData: Blob) => {
    const chunkSizeKB = (audioData.size / 1024).toFixed(2);
    console.log(`📦 PCM Audio chunk ${audioMeta.index}: ${chunkSizeKB} KB`);

    // Verify data integrity
    if (audioData.size !== audioMeta.size) {
      console.warn(`⚠️ Size mismatch chunk ${audioMeta.index}: expected ${audioMeta.size}, got ${audioData.size}`);
    }

    // Skip empty chunks
    if (audioData.size === 0) {
      console.warn(`⚠️ Chunk ${audioMeta.index} is empty, skipping`);
      return;
    }

    try {
      // Create audio chunk - PCM data is directly playable
      const audioChunk: AudioChunk = {
        index: audioMeta.index,
        data: audioData, // Raw PCM data
        received: true
      };

      // Add to queue in order
      setAudioQueue(prev => {
        const newQueue = [...prev, audioChunk].sort((a, b) => a.index - b.index);
        console.log(`🎵 PCM Queue updated: ${newQueue.length} chunks [${newQueue.map(c => c.index).join(', ')}]`);
        return newQueue;
      });

      console.log(`✅ Added PCM chunk ${audioMeta.index} to queue`);

      // Trigger playback if not already playing and we have enough buffer
      if (!isPlayingRef.current && !audioStreamRef.current.isStreaming) {
        setTimeout(() => playAudioQueue(), 50);
      }

    } catch (error) {
      console.error(`Error processing PCM audio chunk ${audioMeta.index}:`, error);
    }
  };

  // Handle legacy audio frames
  const handleLegacyAudioFrame = async (audioData: Blob) => {
    console.log('handleLegacyAudioFrame called - converting to indexed audio');

    try {
      const audioChunk: AudioChunk = {
        index: 1, // Legacy audio is treated as single chunk
        data: audioData, // Use raw data directly
        received: true
      };

      // Add to queue and log length
      setAudioQueue([audioChunk]);
      console.log(`🎵 Queue updated (legacy): 1 chunk [1]`);

    } catch (error) {
      console.error('Error processing legacy audio frame:', error);
    }
  };

  const handleEvent = async (eventName: string, payload: any) => {
    console.log('Received event:', eventName, payload);

    if (eventName === 'audio_format') {
      // Store PCM format information from backend
      audioFormatRef.current = payload;
      console.log('🎵 Audio format received:', payload);
    } else if (eventName === 'message_complete') {
      // Mark current message as complete (no buffering needed for PCM)
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

      // Stop all audio playback and clean up
      stopAllAudio();

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
