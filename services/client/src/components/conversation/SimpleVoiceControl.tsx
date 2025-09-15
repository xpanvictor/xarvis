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
    const [isPushToTalkActive, setIsPushToTalkActive] = useState(false);

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
        const handleListeningModeChange = (eventName: string, payload: any) => {
            if (eventName === 'listening_mode_change') {
                const newMode = payload?.data;
                console.log("🎧 Backend listening mode change:", newMode, payload);
                // We don't need to track active mode locally since push-to-talk handles it
            }
        };

        // Listen for backend events
        webSocketService.on('onEvent', handleListeningModeChange);

        return () => {
            webSocketService.off('onEvent');
        };
    }, []);

    const startPassiveStreaming = async () => {
        try {
            await audioService.requestMicrophoneAccess();

            // Set up continuous audio streaming
            audioService.on('onAudioData', (audioData: ArrayBuffer) => {
                if (isConnected) {
                    webSocketService.sendAudioData(audioData, 16000, 1); // 16kHz, mono
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

    const handlePushToTalk = (isPressed: boolean) => {
        if (!isConnected) return;

        if (isPressed) {
            // Start active listening
            webSocketService.sendListeningControl('start_listening');
            setIsPushToTalkActive(true);
        } else {
            // Stop active listening
            webSocketService.sendListeningControl('stop_listening');
            setIsPushToTalkActive(false);
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
        if (isPushToTalkActive) return 'Push-to-Talk Active';
        if (isPassiveStreaming) return 'Passive Listening';
        return 'Idle';
    };

    const getStatusClass = () => {
        if (!isConnected) return 'disconnected';
        if (isPushToTalkActive) return 'push-to-talk';
        if (isPassiveStreaming) return 'passive';
        return 'idle';
    };

    return (
        <div className={`simple-voice-control ${className}`}>
            {/* Connection Status and Listening Mode */}
            <div className="status-bar">
                {getConnectionIcon()}
                <span className={`status-text ${getStatusClass()}`}>
                    {getStatusText()}
                </span>
            </div>

            {/* Push-to-Talk Button */}
            <button
                className={`push-to-talk ${isPushToTalkActive ? 'active' : ''}`}
                onMouseDown={() => handlePushToTalk(true)}
                onMouseUp={() => handlePushToTalk(false)}
                onMouseLeave={() => handlePushToTalk(false)}
                disabled={!isConnected}
                title="Hold to activate push-to-talk"
            >
                <Mic size={16} />
                <span>Push to Talk</span>
            </button>
        </div>
    );
};

export default SimpleVoiceControl;
