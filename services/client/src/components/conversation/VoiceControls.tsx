import React, { useState, useEffect } from 'react';
import {
    Mic,
    MicOff,
    Volume2,
    VolumeX,
    Circle,
    Pause,
    Play,
    Activity,
    Wifi,
    WifiOff
} from 'lucide-react';
import { useConversationStore } from '../../store';
import audioService from '../../services/audio';
import webSocketService from '../../services/websocket';
import './VoiceControls.css';

interface VoiceControlsProps {
    className?: string;
    onAudioData?: (data: ArrayBuffer) => void;
}

export const VoiceControls: React.FC<VoiceControlsProps> = ({
    className = '',
    onAudioData
}) => {
    const {
        connectionState,
        listeningState,
        isConnected,
        isMuted,
        audioLevel,
        setMuted,
        setAudioLevel
    } = useConversationStore();

    const [isRecording, setIsRecording] = useState(false);
    const [recordingMode, setRecordingMode] = useState<'push-to-talk' | 'toggle'>('push-to-talk');
    const [audioDevices, setAudioDevices] = useState<any[]>([]);
    const [selectedDevice, setSelectedDevice] = useState<string>('');
    const [showSettings, setShowSettings] = useState(false);

    useEffect(() => {
        // Set up audio service event handlers
        audioService.on('onStateChange', (state) => {
            setIsRecording(state === 'recording');
        });

        audioService.on('onVolumeChange', (volume) => {
            setAudioLevel(volume);
        });

        audioService.on('onError', (error) => {
            console.error('Audio service error:', error);
        });

        audioService.on('onDevicesChange', (devices) => {
            setAudioDevices(devices.filter(d => d.kind === 'audioinput'));
        });

        // Load initial settings
        const mutedState = audioService.isMutedState();
        setMuted(mutedState);

        // Get available devices
        audioService.getAudioDevices();

        return () => {
            audioService.cleanup();
        };
    }, [setMuted, setAudioLevel]);

    const handleStartListening = async () => {
        if (!isConnected) {
            console.warn('WebSocket not connected');
            return;
        }

        try {
            await audioService.requestMicrophoneAccess(selectedDevice || undefined);
            webSocketService.sendListeningControl('start_listening');
        } catch (error) {
            console.error('Failed to start listening:', error);
        }
    };

    const handleStopListening = () => {
        if (isConnected) {
            webSocketService.sendListeningControl('stop_listening');
        }
        audioService.stopRecording();
    };

    const handlePushToTalkStart = async () => {
        if (recordingMode === 'push-to-talk') {
            await audioService.startRecording();
            if (isConnected) {
                webSocketService.sendListeningControl('start_listening');
            }
        }
    };

    const handlePushToTalkEnd = async () => {
        if (recordingMode === 'push-to-talk' && isRecording) {
            audioService.stopRecording();
            if (isConnected) {
                webSocketService.sendListeningControl('stop_listening');
            }
        }
    };

    const handleToggleRecording = async () => {
        if (recordingMode === 'toggle') {
            if (isRecording) {
                handleStopListening();
            } else {
                await handleStartListening();
                await audioService.startRecording();
            }
        }
    };

    const handleMuteToggle = () => {
        const newMutedState = !isMuted;
        setMuted(newMutedState);
        audioService.setMuted(newMutedState);
    };

    const getListeningStateIcon = () => {
        switch (listeningState) {
            case 'active':
                return <Activity className="listening-icon active" size={16} />;
            case 'passive':
                return <Circle className="listening-icon passive" size={16} />;
            case 'processing':
                return <Activity className="listening-icon processing" size={16} />;
            default:
                return <Circle className="listening-icon idle" size={16} />;
        }
    };

    const getConnectionIcon = () => {
        if (isConnected) {
            return <Wifi className="connection-icon connected" size={16} />;
        }
        return <WifiOff className="connection-icon disconnected" size={16} />;
    };

    const getVolumeBarHeight = () => {
        return Math.max(2, audioLevel * 100);
    };

    return (
        <div className={`voice-controls ${className}`}>
            {/* Connection Status */}
            <div className="connection-status">
                {getConnectionIcon()}
                <span className={`status-text ${connectionState}`}>
                    {connectionState}
                </span>
            </div>

            {/* Listening State Indicator */}
            <div className="listening-status">
                {getListeningStateIcon()}
                <span className={`listening-text ${listeningState}`}>
                    {listeningState}
                </span>
            </div>

            {/* Audio Level Indicator */}
            <div className="audio-level">
                <div className="volume-bars">
                    {[...Array(10)].map((_, i) => (
                        <div
                            key={i}
                            className={`volume-bar ${i < (audioLevel * 10) ? 'active' : ''
                                }`}
                            style={{
                                height: `${Math.max(2, (i + 1) * 10)}%`,
                                opacity: i < (audioLevel * 10) ? 1 : 0.3
                            }}
                        />
                    ))}
                </div>
            </div>

            {/* Recording Mode Toggle */}
            <div className="recording-mode">
                <button
                    className={`mode-button ${recordingMode === 'push-to-talk' ? 'active' : ''}`}
                    onClick={() => setRecordingMode('push-to-talk')}
                    title="Push to Talk"
                >
                    PTT
                </button>
                <button
                    className={`mode-button ${recordingMode === 'toggle' ? 'active' : ''}`}
                    onClick={() => setRecordingMode('toggle')}
                    title="Toggle Mode"
                >
                    Toggle
                </button>
            </div>

            {/* Main Voice Control Buttons */}
            <div className="main-controls">
                {recordingMode === 'push-to-talk' ? (
                    <button
                        className={`voice-button push-to-talk ${isRecording ? 'recording' : ''}`}
                        onMouseDown={handlePushToTalkStart}
                        onMouseUp={handlePushToTalkEnd}
                        onMouseLeave={handlePushToTalkEnd}
                        onTouchStart={handlePushToTalkStart}
                        onTouchEnd={handlePushToTalkEnd}
                        disabled={!isConnected}
                        title="Hold to speak"
                    >
                        {isRecording ? <Mic className="recording" size={20} /> : <MicOff size={20} />}
                        <span className="button-text">
                            {isRecording ? 'Recording...' : 'Hold to Speak'}
                        </span>
                    </button>
                ) : (
                    <button
                        className={`voice-button toggle ${isRecording ? 'recording' : ''}`}
                        onClick={handleToggleRecording}
                        disabled={!isConnected}
                        title={isRecording ? 'Stop recording' : 'Start recording'}
                    >
                        {isRecording ? <Pause size={20} /> : <Play size={20} />}
                        <span className="button-text">
                            {isRecording ? 'Stop' : 'Start'}
                        </span>
                    </button>
                )}

                {/* Mute Button */}
                <button
                    className={`mute-button ${isMuted ? 'muted' : ''}`}
                    onClick={handleMuteToggle}
                    title={isMuted ? 'Unmute' : 'Mute'}
                >
                    {isMuted ? <VolumeX size={18} /> : <Volume2 size={18} />}
                </button>
            </div>

            {/* Active Listening Controls */}
            {listeningState !== 'idle' && (
                <div className="active-controls">
                    <button
                        className="control-button passive"
                        onClick={handleStartListening}
                        disabled={listeningState === 'passive'}
                        title="Switch to passive listening"
                    >
                        Passive
                    </button>
                    <button
                        className="control-button active"
                        onClick={handleStartListening}
                        disabled={listeningState === 'active'}
                        title="Switch to active listening"
                    >
                        Active
                    </button>
                    <button
                        className="control-button stop"
                        onClick={handleStopListening}
                        title="Stop listening"
                    >
                        Stop
                    </button>
                </div>
            )}

            {/* Settings Panel */}
            {showSettings && (
                <div className="settings-panel">
                    <h4>Audio Settings</h4>

                    <div className="setting-group">
                        <label>Microphone:</label>
                        <select
                            value={selectedDevice}
                            onChange={(e) => setSelectedDevice(e.target.value)}
                        >
                            <option value="">Default</option>
                            {audioDevices.map(device => (
                                <option key={device.deviceId} value={device.deviceId}>
                                    {device.label}
                                </option>
                            ))}
                        </select>
                    </div>

                    <div className="setting-group">
                        <label>
                            <input
                                type="checkbox"
                                checked={audioService.getSettings().echoCancellation}
                                onChange={(e) => audioService.updateSettings({
                                    echoCancellation: e.target.checked
                                })}
                            />
                            Echo Cancellation
                        </label>
                    </div>

                    <div className="setting-group">
                        <label>
                            <input
                                type="checkbox"
                                checked={audioService.getSettings().noiseSuppression}
                                onChange={(e) => audioService.updateSettings({
                                    noiseSuppression: e.target.checked
                                })}
                            />
                            Noise Suppression
                        </label>
                    </div>
                </div>
            )}

            {/* Settings Toggle */}
            <button
                className="settings-toggle"
                onClick={() => setShowSettings(!showSettings)}
                title="Audio settings"
            >
                ⚙️
            </button>
        </div>
    );
};

export default VoiceControls;
