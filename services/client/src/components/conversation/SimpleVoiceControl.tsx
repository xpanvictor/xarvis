import React, { useState, useEffect } from 'react';
import { Mic, MicOff, Wifi, WifiOff } from 'lucide-react';
import { useConversationStore } from '../../store';
import audioService from '../../services/audio';
import webSocketService from '../../services/websocket';
import './SimpleVoiceControl.css';

interface SimpleVoiceControlProps {
    className?: string;
}

export const SimpleVoiceControl: React.FC<SimpleVoiceControlProps> = ({
    className = ''
}) => {
    const {
        connectionState,
        listeningState,
        isConnected
    } = useConversationStore();

    const [isPassiveStreaming, setIsPassiveStreaming] = useState(false);
    const [isActiveMode, setIsActiveMode] = useState(false);

    // Start passive streaming when component mounts and WebSocket is connected
    useEffect(() => {
        if (isConnected && !isPassiveStreaming) {
            startPassiveStreaming();
        } else if (!isConnected && isPassiveStreaming) {
            stopPassiveStreaming();
        }
    }, [isConnected]);

    // Handle listening state changes from backend
    useEffect(() => {
        if (listeningState === 'active') {
            setIsActiveMode(true);
        } else if (listeningState === 'idle') {
            setIsActiveMode(false);
        }
    }, [listeningState]);

    const startPassiveStreaming = async () => {
        try {
            await audioService.requestMicrophoneAccess();

            // Set up continuous audio streaming
            audioService.on('onAudioData', (audioData: ArrayBuffer) => {
                if (isConnected) {
                    webSocketService.sendAudioData(audioData);
                }
            });

            audioService.startStreaming();
            await audioService.startRecording();
            setIsPassiveStreaming(true);
            console.log('Started passive audio streaming');
        } catch (error) {
            console.error('Failed to start passive streaming:', error);
        }
    };

    const stopPassiveStreaming = () => {
        audioService.stopRecording();
        audioService.stopStreaming();
        audioService.off('onAudioData');
        setIsPassiveStreaming(false);
        console.log('Stopped passive audio streaming');
    };

    const handleActiveToggle = () => {
        if (!isConnected) {
            console.warn('WebSocket not connected');
            return;
        }

        if (isActiveMode) {
            // Stop active mode
            webSocketService.sendListeningControl('stop_listening');
        } else {
            // Start active mode
            webSocketService.sendListeningControl('start_listening');
        }
    };

    const getConnectionIcon = () => {
        if (isConnected) {
            return <Wifi className="connection-icon connected" size={14} />;
        }
        return <WifiOff className="connection-icon disconnected" size={14} />;
    };

    const getStatusText = () => {
        if (!isConnected) return 'Disconnected';
        if (isActiveMode) return 'Active Listening';
        if (isPassiveStreaming) return 'Passive Streaming';
        return 'Idle';
    };

    const getStatusClass = () => {
        if (!isConnected) return 'disconnected';
        if (isActiveMode) return 'active';
        if (isPassiveStreaming) return 'passive';
        return 'idle';
    };

    return (
        <div className={`simple-voice-control ${className}`}>
            {/* Connection Status */}
            <div className="status-bar">
                {getConnectionIcon()}
                <span className={`status-text ${getStatusClass()}`}>
                    {getStatusText()}
                </span>
            </div>

            {/* Active Mode Toggle Button */}
            <button
                className={`active-toggle ${isActiveMode ? 'active' : ''}`}
                onClick={handleActiveToggle}
                disabled={!isConnected}
                title={isActiveMode ? 'Stop active listening' : 'Start active listening'}
            >
                {isActiveMode ? (
                    <>
                        <MicOff size={16} />
                        <span>Stop Active</span>
                    </>
                ) : (
                    <>
                        <Mic size={16} />
                        <span>Go Active</span>
                    </>
                )}
            </button>

            {/* Passive Streaming Indicator */}
            {isPassiveStreaming && (
                <div className="streaming-indicator">
                    <div className="pulse-dot"></div>
                    <span>Streaming</span>
                </div>
            )}
        </div>
    );
};

export default SimpleVoiceControl;
