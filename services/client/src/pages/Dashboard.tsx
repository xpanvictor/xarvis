import React, { useEffect, useState } from 'react';
import { MemoryGraph } from '../components/memory/MemoryGraph';
import { RecentMessages } from '../components/conversation/RecentMessages';
import { StreamingMessage } from '../components/conversation/StreamingMessage';
import { SimpleVoiceControl } from '../components/conversation/SimpleVoiceControl';
import { useConversationStore, useUIStore } from '../store';
import { conversationAPI } from '../services/api';
import webSocketService, { ConnectionState } from '../services/websocket';
import audioService from '../services/audio';
import pcmAudioPlayer from '../services/pcmAudioPlayer';
import { Send, Settings, Plus, MessageSquare } from 'lucide-react';
import './Dashboard.css';

export const Dashboard: React.FC = () => {
    const {
        conversation,
        setConversation,
        setLoading,
        setError,
        isConnected,
        connectionState,
        listeningState,
        setConnectionState,
        streamingMessage,
        startStreamingMessage,
        updateStreamingContent,
        completeStreamingMessage,
        setStreamingAudio,
        setStreamingPCMAudio,
        clearStreamingMessage,
        addMessage
    } = useConversationStore();
    const { setSidebarOpen } = useUIStore();
    const [inputMessage, setInputMessage] = useState('');
    const [inputMode, setInputMode] = useState<'text' | 'voice'>('text');
    const [showVoiceControls, setShowVoiceControls] = useState(false);

    // Initialize services and load conversation
    useEffect(() => {
        let mounted = true;

        const initializeApp = async () => {
            try {
                setLoading(true);

                // Load conversation data
                const conv = await conversationAPI.getConversation();
                if (mounted) {
                    setConversation(conv);
                }

                // Initialize WebSocket connection (URL is resolved inside the service)
                webSocketService.connect();

            } catch (error: any) {
                if (mounted) {
                    setError(error.message);
                    console.error('Failed to initialize app:', error);
                }
            } finally {
                if (mounted) {
                    setLoading(false);
                }
            }
        };

        initializeApp();

        return () => {
            mounted = false;
            // Cleanup services on unmount
            webSocketService.disconnect();
            audioService.cleanup();
            pcmAudioPlayer.cleanup();
        };
    }, [setConversation, setLoading, setError]);

    // Handle WebSocket events for streaming
    useEffect(() => {
        // Handle streaming text (existing)
        const handleStreamingText = (content: string, isComplete: boolean) => {
            console.log(`📝 handleStreamingText: content="${content}", isComplete=${isComplete}, hasExistingMessage=${!!streamingMessage}`);

            // Only start streaming if we have actual content
            if (!streamingMessage && content && content.trim()) {
                console.log('🚀 Starting new streaming message');
                startStreamingMessage();
            }

            if (content) {
                updateStreamingContent(content);
            }

            if (isComplete) {
                completeStreamingMessage();
            }
        };

        // Handle streaming audio (existing)
        const handleStreamingAudio = (audioBlob: Blob) => {
            setStreamingAudio(audioBlob);
        };

        // Handle text deltas (like old_ci.tsx)
        const handleTextDelta = (index: number, text: string) => {
            console.log(`📝 Text delta: index=${index}, text="${text}", hasExistingMessage=${!!streamingMessage}`);

            if (index === 1 || !streamingMessage) {
                // Start of new message - clear any existing streaming message
                console.log('🚀 Starting new message from text delta');
                clearStreamingMessage();
                startStreamingMessage();
                updateStreamingContent(text);
            } else {
                // Continue streaming message - append text
                const currentContent = streamingMessage?.content || '';
                const newContent = currentContent + text;
                console.log(`📝 Appending text: "${text}" to existing content: "${currentContent}" = "${newContent}"`);
                updateStreamingContent(newContent);
            }
        };

        // Handle events (like old_ci.tsx)
        const handleEvent = (eventName: string, payload: any) => {
            console.log(`Event: ${eventName}`, payload);

            if (eventName === 'message_complete') {
                // Mark streaming as complete
                completeStreamingMessage();

                // Also try to play audio if we have collected any (fallback)
                setTimeout(() => {
                    if (pcmAudioPlayer.getQueueLength() > 0) {
                        console.log('🎵 Message complete - triggering audio playback as fallback');
                        pcmAudioPlayer.playCollectedAudio();
                    }
                }, 100);
            } else if (eventName === 'audio_format') {
                // Store audio format info for PCM conversion
                console.log('Audio format received:', payload);
                pcmAudioPlayer.setAudioFormat(payload);
            } else if (eventName === 'audio_complete') {
                console.log('🎵 Audio streaming complete - playing collected audio');
                // Play all collected audio at once
                pcmAudioPlayer.playCollectedAudio();
            }
        };

        // Handle PCM audio data - collect for later playback
        const handlePCMAudio = (pcmData: ArrayBuffer) => {
            console.log('Received PCM audio data:', pcmData.byteLength, 'bytes');
            // Just collect the data, don't play yet
            pcmAudioPlayer.addPCMChunk(pcmData);
        };

        // Handle response messages (from websocket response case)
        const handleMessage = (responseData: any) => {
            console.log('📨 Received message response:', responseData);

            if ((responseData.type === 'text' || responseData.type === 'text_delta') && responseData.content) {
                console.log('📝 Processing text response content:', responseData.content);

                // Check if this is a new message or continuation
                if (!streamingMessage) {
                    console.log('🚀 Starting new streaming message from response');
                    startStreamingMessage();
                    updateStreamingContent(responseData.content);
                } else {
                    // For text_delta, append to existing content
                    const currentContent = streamingMessage?.content || '';
                    const newContent = currentContent + responseData.content;
                    console.log(`📝 Appending text delta: "${responseData.content}" to existing: "${currentContent}" = "${newContent}"`);
                    updateStreamingContent(newContent);
                }

                // Only mark as complete if explicitly indicated
                if (responseData.isComplete === true) {
                    console.log('✅ Message marked as complete from response');
                    completeStreamingMessage();
                }
            }
        };

        // Register event handlers
        webSocketService.on('onStreamingText', handleStreamingText);
        webSocketService.on('onStreamingAudio', handleStreamingAudio);
        webSocketService.on('onTextDelta', handleTextDelta);
        webSocketService.on('onEvent', handleEvent);
        webSocketService.on('onPCMAudio', handlePCMAudio);
        webSocketService.on('onMessage', handleMessage);

        // Cleanup
        return () => {
            webSocketService.off('onStreamingText');
            webSocketService.off('onStreamingAudio');
            webSocketService.off('onTextDelta');
            webSocketService.off('onEvent');
            webSocketService.off('onPCMAudio');
            webSocketService.off('onMessage');
        };
    }, [streamingMessage, startStreamingMessage, updateStreamingContent, completeStreamingMessage, setStreamingAudio, clearStreamingMessage]);

    // Handle WebSocket connection state changes
    useEffect(() => {
        const handleConnectionChange = (state: ConnectionState) => {
            setConnectionState(state);
        };

        webSocketService.on('onConnectionStateChange', handleConnectionChange);

        return () => {
            // Clean up listeners when component unmounts
            webSocketService.off('onConnectionStateChange');
        };
    }, [setConnectionState]);

    const handleSendMessage = async () => {
        if (!inputMessage.trim()) return;

        try {
            setLoading(true);
            setError(null);

            // Add user message to recent messages immediately
            const userMessage = {
                id: `user_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`,
                conversation_id: conversation?.id || '',
                user_id: conversation?.owner_id || '',
                text: inputMessage,
                msg_role: 'user' as const,
                timestamp: new Date().toISOString(),
                tags: ['user_input']
            };

            addMessage(userMessage);

            if (inputMode === 'text' && isConnected) {
                // Always use WebSocket for real-time streaming
                clearStreamingMessage(); // Clear any previous streaming message
                webSocketService.sendTextMessage(inputMessage);
            } else {
                // Fallback to API only if WebSocket is not connected
                const response = await conversationAPI.sendMessage({
                    text: inputMessage,
                    timestamp: new Date().toISOString()
                });

                // Add the response message
                addMessage(response);
            }

            setInputMessage('');
        } catch (error) {
            console.error('Failed to send message:', error);
            setError('Failed to send message. Please try again.');
        } finally {
            setLoading(false);
        }
    }; const handleKeyPress = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            handleSendMessage();
        }
    };

    return (
        <div className="dashboard">
            <div className="dashboard-header">
                <div className="header-left">
                    <h1>Conversation Dashboard</h1>
                    <p>Explore your memory graph and recent conversations</p>
                </div>

                <div className="header-actions">
                    <button className="action-button">
                        <Plus size={16} />
                        New Memory
                    </button>
                    <button className="action-button secondary">
                        <Settings size={16} />
                    </button>
                </div>
            </div>

            <div className="dashboard-content">
                {/* Main Memory Graph Area */}
                <div className="memory-section">
                    <div className="section-header">
                        <h2>Memory Graph</h2>
                        <div className="memory-stats">
                            <span className="stat">
                                {conversation?.memories?.length || 0} memories
                            </span>
                            <span className="stat">
                                {conversation?.memories?.filter(m => m.memory_type === 'episodic').length || 0} episodic
                            </span>
                            <span className="stat">
                                {conversation?.memories?.filter(m => m.memory_type === 'semantic').length || 0} semantic
                            </span>
                        </div>
                    </div>

                    <div className="memory-graph-container">
                        <MemoryGraph />
                    </div>
                </div>

                {/* Recent Messages Sidebar */}
                <div className="messages-section">
                    {/* Streaming Message Display */}
                    {streamingMessage && (
                        <StreamingMessage
                            content={streamingMessage.content}
                            isStreaming={streamingMessage.isStreaming}
                            audioBlob={streamingMessage.audioBlob}
                            pcmAudioData={streamingMessage.pcmAudioData}
                            onComplete={() => {
                                // Clear streaming message after completion
                                setTimeout(() => clearStreamingMessage(), 2000);
                            }}
                        />
                    )}

                    <RecentMessages />
                </div>
            </div>

            {/* Input Area */}
            <div className="input-section">
                <div className="input-container">
                    <div className="input-modes">
                        <button
                            className={`mode-button ${inputMode === 'text' ? 'active' : ''}`}
                            onClick={() => setInputMode('text')}
                        >
                            Text
                        </button>
                        <button
                            className={`mode-button ${inputMode === 'voice' ? 'active' : ''}`}
                            onClick={() => setInputMode('voice')}
                        >
                            Voice
                        </button>
                    </div>

                    {inputMode === 'text' ? (
                        <div className="text-input-area">
                            <textarea
                                value={inputMessage}
                                onChange={(e) => setInputMessage(e.target.value)}
                                onKeyPress={handleKeyPress}
                                placeholder="Share your thoughts with Xarvis..."
                                className="message-input"
                                rows={2}
                            />
                            <button
                                className="send-button"
                                onClick={handleSendMessage}
                                disabled={!inputMessage.trim()}
                            >
                                <Send size={18} />
                            </button>
                        </div>
                    ) : (
                        <div className="voice-input-area">
                            <div className="voice-controls-container">
                                <SimpleVoiceControl className="dashboard-voice-controls" />
                            </div>
                        </div>
                    )}
                </div>

                <div className="connection-status">
                    <div className={`status-indicator ${isConnected ? 'connected' : 'disconnected'}`}>
                        <div className="status-dot"></div>
                        <span>{isConnected ? 'Connected' : connectionState}</span>
                    </div>

                    {/* Debug button for testing audio */}
                    <button
                        onClick={() => {
                            console.log('🔧 Debug: Manually triggering audio playback');
                            console.log('🔧 Queue length:', pcmAudioPlayer.getQueueLength());
                            console.log('🔧 Is playing:', pcmAudioPlayer.getIsPlaying());
                            pcmAudioPlayer.playCollectedAudio();
                        }}
                        style={{ marginLeft: '10px', padding: '4px 8px', fontSize: '12px' }}
                    >
                        🔧 Play Audio
                    </button>

                    {listeningState !== 'idle' && (
                        <div className="listening-indicator">
                            <div className={`listening-dot ${listeningState}`}></div>
                            <span className="listening-text">{listeningState}</span>
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
};

export default Dashboard;
