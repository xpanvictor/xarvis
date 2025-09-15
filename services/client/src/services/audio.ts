// Audio service for handling audio input/output with WebSocket integration
export interface AudioDeviceInfo {
    deviceId: string;
    label: string;
    kind: 'audioinput' | 'audiooutput';
}

export interface AudioSettings {
    sampleRate: number;
    channels: number;
    echoCancellation: boolean;
    noiseSuppression: boolean;
    autoGainControl: boolean;
}

export type AudioState = 'idle' | 'recording' | 'processing' | 'playing';

export interface AudioServiceEvents {
    onStateChange: (state: AudioState) => void;
    onVolumeChange: (volume: number) => void;
    onError: (error: string) => void;
    onDevicesChange: (devices: AudioDeviceInfo[]) => void;
    onAudioData: (audioData: ArrayBuffer) => void;
}

class AudioService {
    private mediaRecorder: MediaRecorder | null = null;
    private audioContext: AudioContext | null = null;
    private stream: MediaStream | null = null;
    private analyser: AnalyserNode | null = null;
    private scriptProcessor: ScriptProcessorNode | null = null;
    private volumeCallback: ((volume: number) => void) | null = null;
    private isRecording = false;
    private isMuted = false;
    private audioChunks: Blob[] = [];
    private isStreaming = false;
    private streamingInterval: number | null = null;
    private audioState: AudioState = 'idle';
    private eventHandlers: Partial<AudioServiceEvents> = {};

    // Audio settings
    private settings: AudioSettings = {
        sampleRate: 16000,
        channels: 1,
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true
    };

    constructor() {
        this.initializeAudioContext();
        this.loadSettings();
    }

    // Event handler registration
    on<K extends keyof AudioServiceEvents>(event: K, handler: AudioServiceEvents[K]) {
        this.eventHandlers[event] = handler;
    }

    off<K extends keyof AudioServiceEvents>(event: K) {
        delete this.eventHandlers[event];
    }

    private emit<K extends keyof AudioServiceEvents>(event: K, ...args: Parameters<AudioServiceEvents[K]>) {
        const handler = this.eventHandlers[event];
        if (handler) {
            // @ts-ignore
            handler(...args);
        }
    }

    private updateState(state: AudioState) {
        this.audioState = state;
        this.emit('onStateChange', state);
    }

    private async initializeAudioContext() {
        try {
            // Use browser default sample rate for compatibility
            this.audioContext = new AudioContext();
        } catch (error) {
            console.error('Failed to initialize AudioContext:', error);
            this.emit('onError', 'Failed to initialize audio system');
        }
    }

    private loadSettings() {
        const saved = localStorage.getItem('audioSettings');
        if (saved) {
            try {
                this.settings = { ...this.settings, ...JSON.parse(saved) };
            } catch (error) {
                console.error('Failed to load audio settings:', error);
            }
        }
    }

    private saveSettings() {
        localStorage.setItem('audioSettings', JSON.stringify(this.settings));
    }

    // Device management
    async getAudioDevices(): Promise<AudioDeviceInfo[]> {
        try {
            const devices = await navigator.mediaDevices.enumerateDevices();
            const audioDevices = devices
                .filter(device => device.kind === 'audioinput' || device.kind === 'audiooutput')
                .map(device => ({
                    deviceId: device.deviceId,
                    label: device.label || `${device.kind} ${device.deviceId.slice(0, 8)}`,
                    kind: device.kind as 'audioinput' | 'audiooutput'
                }));

            this.emit('onDevicesChange', audioDevices);
            return audioDevices;
        } catch (error) {
            console.error('Failed to get audio devices:', error);
            this.emit('onError', 'Failed to access audio devices');
            return [];
        }
    }

    // Microphone access and recording
    async requestMicrophoneAccess(deviceId?: string): Promise<boolean> {
        try {
            const constraints: MediaStreamConstraints = {
                audio: {
                    deviceId: deviceId ? { exact: deviceId } : undefined,
                    sampleRate: this.settings.sampleRate,
                    channelCount: this.settings.channels,
                    echoCancellation: this.settings.echoCancellation,
                    noiseSuppression: this.settings.noiseSuppression,
                    autoGainControl: this.settings.autoGainControl
                }
            };

            this.stream = await navigator.mediaDevices.getUserMedia(constraints);

            // Set up audio analysis for volume monitoring
            if (this.audioContext && this.stream) {
                const source = this.audioContext.createMediaStreamSource(this.stream);
                this.analyser = this.audioContext.createAnalyser();
                this.analyser.fftSize = 256;
                source.connect(this.analyser);

                this.startVolumeMonitoring();
            }

            return true;
        } catch (error) {
            console.error('Failed to access microphone:', error);
            this.emit('onError', 'Microphone access denied or failed');
            return false;
        }
    }

    async startRecording(): Promise<boolean> {
        if (!this.stream) {
            const success = await this.requestMicrophoneAccess();
            if (!success) return false;
        }

        try {
            this.audioChunks = [];

            // Ensure audio context exists and is running
            if (!this.audioContext) {
                await this.initializeAudioContext();
            }

            if (!this.audioContext) {
                throw new Error('Failed to initialize audio context');
            }

            // Resume audio context if suspended
            if (this.audioContext.state === 'suspended') {
                await this.audioContext.resume();
            }

            // Create source from stream
            if (!this.stream) {
                throw new Error('No audio stream available');
            }
            const source = this.audioContext.createMediaStreamSource(this.stream);

            // Create script processor for real-time PCM extraction
            this.scriptProcessor = this.audioContext.createScriptProcessor(16384, 1, 1); // 16KB buffer

            let frameCount = 0;
            let audioBuffer: Int16Array[] = [];
            let lastSendTime = Date.now();
            const SEND_INTERVAL_MS = 200; // Send audio every 200ms

            this.scriptProcessor.onaudioprocess = (e) => {
                if (!this.isRecording) return;

                const inputBuffer = e.inputBuffer;
                const inputData = inputBuffer.getChannelData(0);

                // Convert float32 to int16 PCM
                const pcmData = new Int16Array(inputData.length);
                for (let i = 0; i < inputData.length; i++) {
                    pcmData[i] = Math.max(-32768, Math.min(32767, inputData[i] * 32768));
                }

                // Add to buffer
                audioBuffer.push(pcmData);
                frameCount++;

                // Send buffered audio every SEND_INTERVAL_MS
                const now = Date.now();
                if (now - lastSendTime >= SEND_INTERVAL_MS || audioBuffer.length >= 5) {
                    if (audioBuffer.length > 0) {
                        // Combine all buffered frames into one larger chunk
                        const totalSamples = audioBuffer.reduce((sum, chunk) => sum + chunk.length, 0);
                        const combinedPcm = new Int16Array(totalSamples);
                        let offset = 0;

                        for (const chunk of audioBuffer) {
                            combinedPcm.set(chunk, offset);
                            offset += chunk.length;
                        }

                        // Send combined chunk
                        this.emit('onAudioData', combinedPcm.buffer);

                        // Clear buffer
                        audioBuffer = [];
                        lastSendTime = now;
                    }
                }
            };

            // Connect the audio graph
            source.connect(this.scriptProcessor);
            this.scriptProcessor.connect(this.audioContext.destination);

            this.isRecording = true;
            this.updateState('recording');

            console.log('✅ Audio recording started with PCM streaming');
            console.log(`🎤 Audio context sample rate: ${this.audioContext.sampleRate}Hz`);

            return true;
        } catch (error) {
            console.error('Failed to start recording:', error);
            this.emit('onError', 'Failed to start audio recording');
            return false;
        }
    }

    stopRecording() {
        this.isRecording = false;

        // Disconnect and clean up script processor
        if (this.scriptProcessor) {
            try {
                this.scriptProcessor.disconnect();
            } catch (e) {
                // Already disconnected
            }
            this.scriptProcessor = null;
        }

        this.updateState('idle');
        this.stopStreaming();
    }

    // Start continuous audio streaming
    startStreaming() {
        this.isStreaming = true;
        console.log('Started audio streaming mode');
    }

    // Stop continuous audio streaming
    stopStreaming() {
        this.isStreaming = false;
        if (this.streamingInterval) {
            clearInterval(this.streamingInterval);
            this.streamingInterval = null;
        }
        console.log('Stopped audio streaming mode');
    }

    // Process individual audio chunks for streaming (no longer needed with ScriptProcessor)
    // private async processAudioChunk(chunk: Blob) {
    //     try {
    //         const arrayBuffer = await chunk.arrayBuffer();
    //         const pcmData = await this.convertToPCM(arrayBuffer);
    //         this.emit('onAudioData', pcmData);
    //     } catch (error) {
    //         console.error('Failed to process audio chunk:', error);
    //     }
    // }

    private async processRecordedAudio() {
        // Not used with ScriptProcessor approach
        this.updateState('idle');
    }

    private async convertToPCM(audioData: ArrayBuffer): Promise<ArrayBuffer> {
        // With ScriptProcessor, we already have PCM data, so just return it
        return audioData;
    }

    // Audio playback
    async playAudio(audioData: ArrayBuffer, autoplay: boolean = true): Promise<void> {
        if (this.isMuted && !autoplay) {
            console.log('Audio playback skipped - muted');
            return;
        }

        if (!this.audioContext) {
            await this.initializeAudioContext();
        }

        try {
            this.updateState('playing');

            const audioBuffer = await this.audioContext!.decodeAudioData(audioData);
            const source = this.audioContext!.createBufferSource();
            source.buffer = audioBuffer;

            source.connect(this.audioContext!.destination);

            source.onended = () => {
                this.updateState('idle');
            };

            source.start();
        } catch (error) {
            console.error('Failed to play audio:', error);
            this.emit('onError', 'Failed to play audio response');
            this.updateState('idle');
        }
    }

    // Audio playback from URL or Blob
    async playAudioFromUrl(url: string): Promise<void> {
        if (this.isMuted) {
            console.log('Audio playback skipped - muted');
            return;
        }

        try {
            const audio = new Audio(url);
            audio.volume = this.isMuted ? 0 : 1;

            this.updateState('playing');

            audio.onended = () => {
                this.updateState('idle');
            };

            audio.onerror = () => {
                this.emit('onError', 'Failed to play audio from URL');
                this.updateState('idle');
            };

            await audio.play();
        } catch (error) {
            console.error('Failed to play audio from URL:', error);
            this.emit('onError', 'Failed to play audio response');
            this.updateState('idle');
        }
    }

    // Volume monitoring
    private startVolumeMonitoring() {
        if (!this.analyser) return;

        const dataArray = new Uint8Array(this.analyser.frequencyBinCount);

        const updateVolume = () => {
            if (!this.analyser) return;

            this.analyser.getByteFrequencyData(dataArray);

            // Calculate RMS volume
            let sum = 0;
            for (let i = 0; i < dataArray.length; i++) {
                sum += dataArray[i] * dataArray[i];
            }
            const rms = Math.sqrt(sum / dataArray.length);
            const volume = rms / 255; // Normalize to 0-1

            this.emit('onVolumeChange', volume);

            if (this.isRecording) {
                requestAnimationFrame(updateVolume);
            }
        };

        updateVolume();
    }

    // Settings management
    updateSettings(newSettings: Partial<AudioSettings>) {
        this.settings = { ...this.settings, ...newSettings };
        this.saveSettings();
    }

    getSettings(): AudioSettings {
        return { ...this.settings };
    }

    // Mute control
    setMuted(muted: boolean) {
        this.isMuted = muted;
        localStorage.setItem('audioMuted', muted.toString());
    }

    isMutedState(): boolean {
        const saved = localStorage.getItem('audioMuted');
        if (saved !== null) {
            this.isMuted = saved === 'true';
        }
        return this.isMuted;
    }

    // State getters
    getState(): AudioState {
        return this.audioState;
    }

    getIsRecording(): boolean {
        return this.isRecording;
    }

    // Cleanup
    cleanup() {
        if (this.mediaRecorder && this.isRecording) {
            this.mediaRecorder.stop();
        }

        // Clean up script processor
        if (this.scriptProcessor) {
            try {
                this.scriptProcessor.disconnect();
            } catch (e) {
                // Already disconnected
            }
            this.scriptProcessor = null;
        }

        if (this.stream) {
            this.stream.getTracks().forEach(track => track.stop());
            this.stream = null;
        }

        if (this.audioContext) {
            this.audioContext.close();
            this.audioContext = null;
        }

        this.updateState('idle');
    }
}

// Create and export singleton instance
export const audioService = new AudioService();
export default audioService;

// Export the audio context for use by other services
export const getAudioContext = () => audioService['audioContext'];
