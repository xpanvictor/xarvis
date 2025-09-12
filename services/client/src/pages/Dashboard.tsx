import React, { useEffect, useState } from 'react';
import { MemoryGraph } from '../components/memory/MemoryGraph';
import { RecentMessages } from '../components/conversation/RecentMessages';
import { useConversationStore, useUIStore } from '../store';
import { conversationAPI } from '../services/api';
import { Send, Mic, MicOff, Settings, Plus } from 'lucide-react';
import './Dashboard.css';

export const Dashboard: React.FC = () => {
    const { conversation, setConversation, setLoading, setError } = useConversationStore();
    const { setSidebarOpen } = useUIStore();
    const [inputMessage, setInputMessage] = useState('');
    const [isConnected, setIsConnected] = useState(false);
    const [inputMode, setInputMode] = useState<'text' | 'voice'>('text');

    // Load conversation data on mount
    useEffect(() => {
        const loadConversation = async () => {
            try {
                setLoading(true);
                const conv = await conversationAPI.getConversation();
                setConversation(conv);
            } catch (error: any) {
                setError(error.message);
                console.error('Failed to load conversation:', error);
            } finally {
                setLoading(false);
            }
        };

        loadConversation();
    }, [setConversation, setLoading, setError]);

    const handleSendMessage = async () => {
        if (!inputMessage.trim()) return;

        try {
            await conversationAPI.sendMessage({
                text: inputMessage,
                timestamp: new Date().toISOString(),
            });

            setInputMessage('');

            // Reload conversation to get updated messages
            const updatedConv = await conversationAPI.getConversation();
            setConversation(updatedConv);
        } catch (error: any) {
            setError(error.message);
            console.error('Failed to send message:', error);
        }
    };

    const handleKeyPress = (e: React.KeyboardEvent) => {
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
                            <button className="voice-button">
                                <Mic size={20} />
                                <span>Tap to speak</span>
                            </button>
                            <div className="voice-status">
                                Ready to listen
                            </div>
                        </div>
                    )}
                </div>

                <div className="connection-status">
                    <div className={`status-indicator ${isConnected ? 'connected' : 'disconnected'}`}>
                        <div className="status-dot"></div>
                        <span>{isConnected ? 'Connected' : 'Disconnected'}</span>
                    </div>
                </div>
            </div>
        </div>
    );
};

export default Dashboard;
