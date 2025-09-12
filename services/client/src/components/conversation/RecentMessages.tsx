import React, { useRef, useEffect } from 'react';
import { User, Bot, Clock, MessageCircle } from 'lucide-react';
import { useConversationStore } from '../../store';
import { Message } from '../../services/api';
import ReactMarkdown from 'react-markdown';
import './RecentMessages.css';

interface RecentMessagesProps {
    className?: string;
}

export const RecentMessages: React.FC<RecentMessagesProps> = ({ className = '' }) => {
    const { recentMessages } = useConversationStore();
    const messagesEndRef = useRef<HTMLDivElement>(null);

    // Auto-scroll to the latest message when messages update
    useEffect(() => {
        if (messagesEndRef.current) {
            messagesEndRef.current.scrollIntoView({ behavior: 'smooth' });
        }
    }, [recentMessages]);

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

        // Less than 24 hours
        if (diff < 86400000) {
            const hours = Math.floor(diff / 3600000);
            return `${hours}h ago`;
        }

        // More than 24 hours
        return date.toLocaleDateString();
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

    if (recentMessages.length === 0) {
        return (
            <div className={`recent-messages-empty ${className}`}>
                <div className="empty-state">
                    <MessageCircle size={24} opacity={0.5} />
                    <p>No recent messages</p>
                </div>
            </div>
        );
    }

    return (
        <div className={`recent-messages ${className}`}>
            <div className="recent-messages-header">
                <h3>Recent Messages</h3>
                <span className="message-count">{recentMessages.length}</span>
            </div>

            <div className="messages-timeline">
                {recentMessages.map((message, index) => (
                    <div key={message.id} className="timeline-item">
                        <div className="timeline-connector">
                            <div
                                className="timeline-dot"
                                style={{ backgroundColor: getRoleColor(message.msg_role) }}
                            >
                                {getRoleIcon(message.msg_role)}
                            </div>
                            {index < recentMessages.length - 1 && (
                                <div className="timeline-line" />
                            )}
                        </div>

                        <div className="message-card">
                            <div className="message-header">
                                <span className={`role-badge ${message.msg_role}`}>
                                    {message.msg_role}
                                </span>
                                <span className="timestamp">
                                    {formatTimestamp(message.timestamp)}
                                </span>
                            </div>

                            <div className="message-content">
                                {message.msg_role === 'assistant' ? (
                                    <ReactMarkdown className="markdown-content">
                                        {message.text.length > 150
                                            ? message.text.substring(0, 150) + '...'
                                            : message.text
                                        }
                                    </ReactMarkdown>
                                ) : (
                                    <p className="text-content">
                                        {message.text.length > 150
                                            ? message.text.substring(0, 150) + '...'
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
                    View Full Conversation
                </button>
            </div>
        </div>
    );
};

export default RecentMessages;
