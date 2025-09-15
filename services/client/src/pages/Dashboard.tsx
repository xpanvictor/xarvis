import React, { useEffect, useState, useRef } from 'react';
import { MemoryGraph } from '../components/memory/MemoryGraph';
import { ChatThread } from '../components/conversation/ChatThread';
import { SimpleVoiceControl } from '../components/conversation/SimpleVoiceControl';
import { useConversationStore, useUIStore } from '../store';
import { conversationAPI } from '../services/api';
import webSocketService, { ConnectionState } from '../services/websocket';
import audioService from '../services/audio';
import pcmAudioPlayer from '../services/pcmAudioPlayer';
import { streamingPCMAudioPlayer } from '../services/streamingAudioPlayer';
import { Send, Settings, Plus, MessageSquare, Volume2, VolumeX } from 'lucide-react';
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
        isMuted,
        isSessionProcessing,
        setConnectionState,
        setSessionProcessing,
        streamingMessage,
        startStreamingMessage,
        updateStreamingContent,
        completeStreamingMessage,
        setStreamingAudio,
        setStreamingPCMAudio,
        clearStreamingMessage,
        addMessage,
        setMuted
    } = useConversationStore();
    const { setSidebarOpen } = useUIStore();
    const [inputMessage, setInputMessage] = useState('');
    const [inputMode, setInputMode] = useState<'text' | 'voice'>('text');
    const [showVoiceControls, setShowVoiceControls] = useState(false);
    const [viewMode, setViewMode] = useState<'split' | 'chat' | 'memory'>('split');

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

                // Sync mute state from persisted audio service setting
                const savedMuted = audioService.isMutedState();
                setMuted(savedMuted);

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
            streamingPCMAudioPlayer.cleanup();
        };
    }, [setConversation, setLoading, setError]);

    // Handle WebSocket events for streaming
    useEffect(() => {
        // Prevent duplicate playback per response
        const audioPlayTriggeredRef = { current: false } as React.MutableRefObject<boolean>;

        // Handle streaming text (existing)
        const handleStreamingText = (content: string, isComplete: boolean) => {
            console.log(`📝 handleStreamingText: content="${content}", isComplete=${isComplete}, hasExistingMessage=${!!streamingMessage}, sessionProcessing=${isSessionProcessing}`);

            // Skip empty content unless it's a completion signal
            if (!content && !isComplete) {
                console.log('📝 Skipping empty streaming text content');
                return;
            }

            // If we have content and there's an existing streaming message, always update it
            if (content && streamingMessage) {
                console.log('📝 Updating existing streaming message');
                // Accumulate content
                const currentContent = streamingMessage.content || '';
                const newContent = currentContent + content;
                updateStreamingContent(newContent);
            }
            // If we have content and no existing message, only start new message if session not processing
            else if (content && content.trim() && !streamingMessage) {
                if (isSessionProcessing) {
                    console.log('⚠️ Blocking new message start - session still processing');
                    return;
                }
                console.log('🚀 Starting new streaming message');
                setSessionProcessing(true); // Mark session as processing

                // Clear any previous audio collection when starting new session
                console.log('🎵 Stopping previous streaming audio for new session');
                streamingPCMAudioPlayer.stop();

                startStreamingMessage();
                updateStreamingContent(content);
            }

            if (isComplete) {
                console.log('📝 Completion signal received - completing streaming message');
                if (streamingMessage && streamingMessage.isStreaming) {
                    completeStreamingMessage();
                } else {
                    console.log('📝 Completion signal received but no active streaming message to complete');
                }
                // NOTE: Don't end session processing here - wait for message_complete
            }
        };

        // Handle streaming audio (existing)
        const handleStreamingAudio = (audioBlob: Blob) => {
            setStreamingAudio(audioBlob);
        };

        // Handle text deltas (like old_ci.tsx)
        const handleTextDelta = (index: number, text: string) => {
            console.log(`📝 Text delta: index=${index}, text="${text}", hasExistingMessage=${!!streamingMessage}`);

            // Skip empty text deltas
            if (!text || !text.trim()) {
                console.log('📝 Skipping empty text delta');
                return;
            }

            // If there's already a streaming message with the same content, skip
            if (streamingMessage && streamingMessage.content === text) {
                console.log('📝 Skipping text delta - content already matches streaming message');
                return;
            }

            // Only start new message if there's no existing streaming message
            // (handleStreamingText should have handled this already)
            if (!streamingMessage && text && text.trim()) {
                console.log('🚀 Starting new message from text delta (fallback)');
                startStreamingMessage();
                updateStreamingContent(text);
            } else if (streamingMessage && text) {
                // Continue streaming message - append text
                const currentContent = streamingMessage.content || '';
                const newContent = currentContent + text;
                console.log(`📝 Appending text delta: "${text}" to existing: "${currentContent}" = "${newContent}"`);
                updateStreamingContent(newContent);
            }
        };

        // Handle events (like old_ci.tsx)
        const handleEvent = (eventName: string, payload: any) => {
            console.log(`Event: ${eventName}`, payload);

            if (eventName === 'text_complete') {
                // ONLY text_complete should stop UI and push to queue
                console.log('📝 TEXT_COMPLETE event - completing streaming message');
                if (streamingMessage && streamingMessage.isStreaming) {
                    completeStreamingMessage();
                }
            } else if (eventName === 'message_complete') {
                // Session is now complete - allow new messages
                console.log('📝 MESSAGE_COMPLETE event - session finished, allowing new messages');
                setSessionProcessing(false); // Allow new messages to start
                // Note: Streaming audio player handles playback continuously, no fallback needed
            } else if (eventName === 'audio_format') {
                // Store audio format info for PCM conversion
                console.log('🎵 Audio format received:', payload);
                // Set format for streaming audio player
                streamingPCMAudioPlayer.setAudioFormat(payload.sampleRate || 22050, payload.channels || 1);
                // New stream beginning; allow a new play trigger for this message
                audioPlayTriggeredRef.current = false;
            } else if (eventName === 'audio_complete') {
                // Audio streaming complete - for streaming player, this might be a signal to finish playback
                console.log('🎵 AUDIO_COMPLETE event - audio streaming finished');
                // Note: Streaming audio player handles playback continuously, so we don't need to trigger playback here
                // This event just indicates the stream is complete
            }
        };

        // Handle PCM audio data - stream for real-time playback
        const handlePCMAudio = (pcmData: ArrayBuffer) => {
            console.log(`🎵 PCM audio received: ${pcmData.byteLength} bytes, session processing: ${isSessionProcessing}`);
            // Only skip audio if muted - allow audio to play even after text completion
            const muted = useConversationStore.getState().isMuted;
            if (!muted) {
                streamingPCMAudioPlayer.addPCMChunk(pcmData);
                console.log(`🎵 Streaming PCM chunk added, buffer health: ${streamingPCMAudioPlayer.getBufferHealth().toFixed(2)}`);
            } else {
                console.log('🔇 Skipping PCM audio streaming due to mute');
            }
        };

        // Handle response messages (from websocket response case)
        const handleMessage = (responseData: any) => {
            console.log('📨 Received message response:', responseData);

            // Skip text_delta and text_complete types as they're handled by streaming handlers
            if (responseData.type === 'text_delta' || responseData.type === 'text_complete') {
                console.log('📝 Skipping text_delta/text_complete in handleMessage - handled by streaming');
                return;
            }

            if ((responseData.type === 'text') && responseData.content) {
                console.log('📝 Processing text response content:', responseData.content);

                // Check if this is a new message or continuation
                if (!streamingMessage) {
                    console.log('🚀 Starting new streaming message from response');
                    startStreamingMessage();
                    updateStreamingContent(responseData.content);
                } else {
                    // For text, append to existing content
                    const currentContent = streamingMessage?.content || '';
                    const newContent = currentContent + responseData.content;
                    console.log(`📝 Appending text: "${responseData.content}" to existing: "${currentContent}" = "${newContent}"`);
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
                // If a previous streaming bubble exists and hasn't finalized, finalize it
                const sm = useConversationStore.getState().streamingMessage;
                if (sm && sm.isStreaming && sm.content) {
                    completeStreamingMessage();
                }
                // Reset session processing for new request
                setSessionProcessing(false);
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
                    <span className="stat" title={isMuted ? 'Muted' : 'Sound on'} style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
                        {isMuted ? <VolumeX size={14} /> : <Volume2 size={14} />}
                        {isMuted ? 'Muted' : 'Sound on'}
                    </span>
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
                <div className={`memory-section ${viewMode === 'memory' ? 'expanded' : viewMode === 'chat' ? 'collapsed' : ''}`}>
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
                            {viewMode === 'memory' ? (
                                <button className={`action-button secondary`} onClick={() => setViewMode('split')} title="Split view" style={{ marginLeft: 8 }}>
                                    Split ◫
                                </button>
                            ) : (
                                <button className={`action-button secondary`} onClick={() => setViewMode('memory')} title="Expand memory" style={{ marginLeft: 8 }}>
                                    Expand ⤢
                                </button>
                            )}
                        </div>
                    </div>

                    <div className="memory-graph-container">
                        <MemoryGraph />
                    </div>
                </div>

                {/* Unified Chat Thread */}
                <div className={`messages-section ${viewMode === 'chat' ? 'expanded' : viewMode === 'memory' ? 'collapsed' : ''}`}>
                    <div className="section-header">
                        <h2>Conversation</h2>
                        <div>
                            {viewMode === 'chat' ? (
                                <button className={`action-button secondary`} onClick={() => setViewMode('split')} title="Split view">
                                    Split ◫
                                </button>
                            ) : (
                                <button className={`action-button secondary`} onClick={() => setViewMode('chat')} title="Expand chat">
                                    Chat ⤢
                                </button>
                            )}
                        </div>
                    </div>
                    <ChatThread />
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
                    {/* Mute toggle controlling autoplay */}
                    <button
                        onClick={() => {
                            const curr = useConversationStore.getState().isMuted;
                            useConversationStore.getState().setMuted(!curr);
                            try { audioService.setMuted(!curr); } catch { }
                        }}
                        className={`mute-toggle ${isMuted ? 'muted' : ''}`}
                        title={isMuted ? 'Unmute' : 'Mute'}
                        style={{ marginLeft: '10px', padding: '4px 8px', fontSize: '12px' }}
                    >
                        {isMuted ? 'Unmute 🔈' : 'Mute 🔇'}
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
