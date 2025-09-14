import React, { useState, useEffect, useRef } from 'react';
import { Bot, Volume2, VolumeX } from 'lucide-react';
import pcmAudioPlayer from '../../services/pcmAudioPlayer';
import './StreamingMessage.css';

interface StreamingMessageProps {
    content: string;
    isStreaming: boolean;
    audioBlob?: Blob | null;
    pcmAudioData?: ArrayBuffer | null;  // Add support for raw PCM data
    onComplete?: () => void;
    className?: string;
}

export const StreamingMessage: React.FC<StreamingMessageProps> = ({
    content,
    isStreaming,
    audioBlob,
    pcmAudioData,
    onComplete,
    className = ''
}) => {
    const [displayedContent, setDisplayedContent] = useState('');
    const [isAudioPlaying, setIsAudioPlaying] = useState(false);
    const [hasAudio, setHasAudio] = useState(false);
    const contentRef = useRef<HTMLDivElement>(null);

    // Mark presence of audio if either PCM or blob is provided (UI hint only)
    useEffect(() => {
        if (pcmAudioData || audioBlob) {
            setHasAudio(true);
        }
    }, [pcmAudioData, audioBlob]);

    // Stream text character by character for typing effect
    useEffect(() => {
        console.log('StreamingMessage: content changed:', { content, isStreaming, contentLength: content.length });

        // Only update if we have actual content or explicitly clearing
        if (!content && isStreaming) {
            // Don't update displayed content if we're streaming but have no content yet
            return;
        }

        if (!content) {
            setDisplayedContent('');
            return;
        }

        if (isStreaming) {
            // Show content immediately for now, but add typing effect
            setDisplayedContent(content);
            console.log('StreamingMessage: displaying content:', content);
        } else {
            // Show all content immediately when not streaming
            setDisplayedContent(content);
            if (onComplete) {
                onComplete();
            }
        }
    }, [content, isStreaming, onComplete]);

    // Auto-scroll to latest content
    useEffect(() => {
        if (contentRef.current) {
            contentRef.current.scrollTop = contentRef.current.scrollHeight;
        }
    }, [displayedContent]);

    const handleAudioToggle = () => {
        // We use a single-flow buffered player: play all collected chunks together
        if (isAudioPlaying) {
            pcmAudioPlayer.stopAllAudio();
            setIsAudioPlaying(false);
            return;
        }

        // Trigger playback of whatever has been collected for this response
        pcmAudioPlayer.playCollectedAudio();
        setIsAudioPlaying(true);
        // Best-effort: reset flag when playback ends (polling the service status)
        setTimeout(() => {
            const interval = setInterval(() => {
                if (!pcmAudioPlayer.getIsPlaying()) {
                    clearInterval(interval);
                    setIsAudioPlaying(false);
                }
            }, 200);
        }, 200);
    };

    if (!content && !isStreaming) {
        return null;
    }

    return (
        <div className={`streaming-message ${className}`}>
            <div className="message-header">
                <div className="message-avatar">
                    <Bot size={16} />
                </div>
                <div className="message-meta">
                    <span className="message-role">Assistant</span>
                    <div className="message-controls">
                        <button
                            className={`audio-toggle ${isAudioPlaying ? 'playing' : ''}`}
                            onClick={handleAudioToggle}
                            disabled={!hasAudio}
                            title={isAudioPlaying ? 'Stop Audio' : 'Play Audio'}
                        >
                            {isAudioPlaying ? <VolumeX size={14} /> : <Volume2 size={14} />}
                        </button>
                    </div>
                </div>
            </div>

            <div className="message-content" ref={contentRef}>
                <div className="text-content">
                    {displayedContent}
                    {isStreaming && (
                        <span className="typing-cursor">|</span>
                    )}
                </div>
            </div>

            {/* Audio status indicator (no longer need HTML audio element) */}
            {hasAudio && (
                <div className="audio-status">
                    <span>{isAudioPlaying ? 'Playing audio...' : 'Audio ready'}</span>
                </div>
            )}

            {isStreaming && (
                <div className="streaming-indicator">
                    <div className="pulse-dots">
                        <span></span>
                        <span></span>
                        <span></span>
                    </div>
                </div>
            )}
        </div>
    );
};

export default StreamingMessage;
