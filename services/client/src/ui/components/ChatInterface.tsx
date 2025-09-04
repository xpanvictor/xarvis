import React, { useState, useRef, useEffect, useCallback } from 'react';
import { Send, User, Bot } from 'lucide-react';
import ReactMarkdown from 'react-markdown';

interface Message {
  id: string;
  content: string;
  role: 'user' | 'assistant';
  timestamp: Date;
  audioChunks?: ArrayBuffer[];
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
  const audioChunksRef = useRef<ArrayBuffer[]>([]);
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

  const handleAudioFrame = (audioData: ArrayBuffer) => {
    audioChunksRef.current.push(audioData);

    // Update current message with audio chunks
    setMessages(prev => prev.map(msg =>
      msg.id === currentMessageIdRef.current
        ? { ...msg, audioChunks: [...audioChunksRef.current] }
        : msg
    ));
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

      // Play audio if available
      if (audioChunksRef.current.length > 0) {
        playAudioChunks(audioChunksRef.current);
      }

      setIsLoading(false);
      currentMessageRef.current = '';
      audioChunksRef.current = [];
    }
  };

  const playAudioChunks = async (chunks: ArrayBuffer[]) => {
    try {
      // Combine all audio chunks into a single blob
      const audioBlob = new Blob(chunks, { type: 'audio/wav' });
      const audioUrl = URL.createObjectURL(audioBlob);
      const audio = new Audio(audioUrl);

      await audio.play();

      // Clean up the URL after playing
      audio.onended = () => {
        URL.revokeObjectURL(audioUrl);
      };
    } catch (error) {
      console.error('Error playing audio:', error);
    }
  };

  useEffect(() => {
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
              {message.audioChunks && message.audioChunks.length > 0 && !message.isStreaming && (
                <div className="message-audio">
                  <button
                    onClick={() => playAudioChunks(message.audioChunks!)}
                    className="play-audio-button"
                  >
                    🔊 Play Audio Response
                  </button>
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
