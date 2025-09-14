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
            this.audioContext = new AudioContext({
                sampleRate: this.settings.sampleRate
            });
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
            this.mediaRecorder = new MediaRecorder(this.stream!, {
                mimeType: 'audio/webm; codecs=opus'
            });

            this.mediaRecorder.ondataavailable = (event) => {
                if (event.data.size > 0) {
                    this.audioChunks.push(event.data);

                    // For streaming mode, process and emit audio data immediately
                    if (this.isStreaming) {
                        this.processAudioChunk(event.data);
                    }
                }
            };

            this.mediaRecorder.onstop = () => {
                this.processRecordedAudio();
            };

            this.mediaRecorder.start(100); // Collect data every 100ms
            this.isRecording = true;
            this.updateState('recording');

            return true;
        } catch (error) {
            console.error('Failed to start recording:', error);
            this.emit('onError', 'Failed to start audio recording');
            return false;
        }
    }

    stopRecording() {
        if (this.mediaRecorder && this.isRecording) {
            this.mediaRecorder.stop();
            this.isRecording = false;
            this.updateState('processing');
        }
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

    // Process individual audio chunks for streaming
    private async processAudioChunk(chunk: Blob) {
        try {
            const arrayBuffer = await chunk.arrayBuffer();
            const pcmData = await this.convertToPCM(arrayBuffer);
            this.emit('onAudioData', pcmData);
        } catch (error) {
            console.error('Failed to process audio chunk:', error);
        }
    }

    private async processRecordedAudio() {
        if (this.audioChunks.length === 0) {
            this.updateState('idle');
            return;
        }

        try {
            const audioBlob = new Blob(this.audioChunks, { type: 'audio/webm' });
            const arrayBuffer = await audioBlob.arrayBuffer();

            // Convert to PCM if needed
            const pcmData = await this.convertToPCM(arrayBuffer);

            // Send via WebSocket (this will be called from the component)
            this.emit('onStateChange', 'idle');

            return pcmData;
        } catch (error) {
            console.error('Failed to process audio:', error);
            this.emit('onError', 'Failed to process recorded audio');
            this.updateState('idle');
        }
    }

    private async convertToPCM(audioData: ArrayBuffer): Promise<ArrayBuffer> {
        if (!this.audioContext) {
            await this.initializeAudioContext();
        }

        if (!this.audioContext) {
            throw new Error('AudioContext failed to initialize');
        }

        try {
            const audioBuffer = await this.audioContext.decodeAudioData(audioData);
            const pcmData = new Float32Array(audioBuffer.length);
            audioBuffer.copyFromChannel(pcmData, 0);

            // Convert to 16-bit PCM
            const pcm16 = new Int16Array(pcmData.length);
            for (let i = 0; i < pcmData.length; i++) {
                pcm16[i] = Math.max(-32768, Math.min(32767, pcmData[i] * 32768));
            }

            return pcm16.buffer;
        } catch (error) {
            console.error('Failed to convert to PCM:', error);
            throw error;
        }
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
