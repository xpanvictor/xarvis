import React, { useRef, useEffect, useMemo } from 'react';
import { User, Bot, Clock, MessageCircle, Wifi, WifiOff } from 'lucide-react';
import { useConversationStore } from '../../store';
import { Message } from '../../services/api';
import ReactMarkdown from 'react-markdown';
import './RecentMessages.css';

interface RecentMessagesProps {
    className?: string;
    maxMessages?: number;
    showConnectionStatus?: boolean;
}

export const RecentMessages: React.FC<RecentMessagesProps> = ({
    className = '',
    maxMessages = 10,
    showConnectionStatus = true
}) => {
    const { recentMessages, isConnected, connectionState, listeningState } = useConversationStore();
    const messagesEndRef = useRef<HTMLDivElement>(null);

    // Sort messages by timestamp (newest first for display, but maintain chronological order)
    const sortedMessages = useMemo(() => {
        return [...recentMessages]
            .sort((a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime())
            .slice(-maxMessages); // Take the most recent messages
    }, [recentMessages, maxMessages]);

    // Auto-scroll to the latest message when messages update
    useEffect(() => {
        if (messagesEndRef.current) {
            messagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
        }
    }, [sortedMessages]);

    const formatTimestamp = (timestamp: string) => {
        const date = new Date(timestamp);
        const now = new Date();
        const diff = now.getTime() - date.getTime();

        // Less than 1 minute
        if (diff < 60000) {
            return 'Just now';
        }

        // Less than 1 hour
        if (diff < 3600000) {
            const minutes = Math.floor(diff / 60000);
            return `${minutes}m ago`;
        }

        // Less than 1 day
        if (diff < 86400000) {
            const hours = Math.floor(diff / 3600000);
            return `${hours}h ago`;
        }

        // More than 1 day but less than 1 week
        if (diff < 604800000) {
            const days = Math.floor(diff / 86400000);
            return `${days}d ago`;
        }

        // Format as date
        return date.toLocaleDateString(undefined, {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    };

    const getRoleIcon = (role: Message['msg_role']) => {
        switch (role) {
            case 'user':
                return <User size={14} />;
            case 'assistant':
                return <Bot size={14} />;
            case 'system':
                return <Clock size={14} />;
            default:
                return <MessageCircle size={14} />;
        }
    };

    const getRoleColor = (role: Message['msg_role']) => {
        switch (role) {
            case 'user':
                return '#667eea';
            case 'assistant':
                return '#64ffda';
            case 'system':
                return '#888';
            default:
                return '#888';
        }
    };

    if (sortedMessages.length === 0) {
        return (
            <div className={`recent-messages-empty ${className}`}>
                {showConnectionStatus && (
                    <div className="connection-indicator">
                        {isConnected ? (
                            <div className="status-connected">
                                <Wifi size={16} />
                                <span>Connected</span>
                                {listeningState !== 'idle' && (
                                    <span className={`listening-badge ${listeningState}`}>
                                        {listeningState}
                                    </span>
                                )}
                            </div>
                        ) : (
                            <div className="status-disconnected">
                                <WifiOff size={16} />
                                <span>{connectionState}</span>
                            </div>
                        )}
                    </div>
                )}
                <div className="empty-state">
                    <MessageCircle size={24} opacity={0.5} />
                    <p>No recent messages</p>
                    <small>Start a conversation to see messages here</small>
                </div>
            </div>
        );
    }

    return (
        <div className={`recent-messages ${className}`}>
            <div className="recent-messages-header">
                <div className="header-title">
                    <h3>Recent Messages</h3>
                    <span className="message-count">{sortedMessages.length}</span>
                </div>

                {showConnectionStatus && (
                    <div className="connection-indicator">
                        {isConnected ? (
                            <div className="status-connected">
                                <Wifi size={14} />
                                <span>Live</span>
                                {listeningState !== 'idle' && (
                                    <span className={`listening-badge ${listeningState}`}>
                                        {listeningState}
                                    </span>
                                )}
                            </div>
                        ) : (
                            <div className="status-disconnected">
                                <WifiOff size={14} />
                                <span>{connectionState}</span>
                            </div>
                        )}
                    </div>
                )}
            </div>

            <div className="messages-timeline">
                {sortedMessages.map((message, index) => (
                    <div key={`${message.id}-${message.timestamp}`} className="timeline-item">
                        <div className="timeline-connector">
                            <div
                                className="timeline-dot"
                                style={{ backgroundColor: getRoleColor(message.msg_role) }}
                            >
                                {getRoleIcon(message.msg_role)}
                            </div>
                            {index < sortedMessages.length - 1 && (
                                <div className="timeline-line" />
                            )}
                        </div>

                        <div className="message-card">
                            <div className="message-header">
                                <span className={`role-badge ${message.msg_role}`}>
                                    {message.msg_role}
                                </span>
                                <span className="timestamp" title={new Date(message.timestamp).toLocaleString()}>
                                    {formatTimestamp(message.timestamp)}
                                </span>
                            </div>

                            <div className="message-content">
                                {message.msg_role === 'assistant' ? (
                                    <ReactMarkdown className="markdown-content">
                                        {message.text.length > 200
                                            ? message.text.substring(0, 200) + '...'
                                            : message.text
                                        }
                                    </ReactMarkdown>
                                ) : (
                                    <p className="text-content">
                                        {message.text.length > 200
                                            ? message.text.substring(0, 200) + '...'
                                            : message.text
                                        }
                                    </p>
                                )}
                            </div>

                            {message.tags && message.tags.length > 0 && (
                                <div className="message-tags">
                                    {message.tags.slice(0, 3).map((tag, tagIndex) => (
                                        <span key={tagIndex} className="tag">
                                            {tag}
                                        </span>
                                    ))}
                                    {message.tags.length > 3 && (
                                        <span className="tag-more">
                                            +{message.tags.length - 3}
                                        </span>
                                    )}
                                </div>
                            )}
                        </div>
                    </div>
                ))}
                {/* Invisible div at the end for auto-scroll */}
                <div ref={messagesEndRef} />
            </div>

            <div className="recent-messages-footer">
                <button className="view-all-button">
                    View Full Conversation ({recentMessages.length} total)
                </button>
            </div>
        </div>
    );
};

export default RecentMessages;
