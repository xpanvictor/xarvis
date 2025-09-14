// Audio Player Service for handling PCM audio streams
class AudioPlayerService {
    private audioContext: AudioContext | null = null;
    private sourceNode: AudioBufferSourceNode | null = null;
    private isPlaying = false;
    private audioQueue: Float32Array[] = [];
    private isProcessing = false;

    constructor() {
        this.initializeAudioContext();
    }

    private async initializeAudioContext() {
        try {
            this.audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();

            // Resume context if it's suspended (required by browser autoplay policies)
            if (this.audioContext.state === 'suspended') {
                await this.audioContext.resume();
            }
        } catch (error) {
            console.error('Failed to initialize audio context:', error);
        }
    }

    private async ensureAudioContext() {
        if (!this.audioContext) {
            await this.initializeAudioContext();
        }

        if (this.audioContext && this.audioContext.state === 'suspended') {
            await this.audioContext.resume();
        }
    }

    // Convert PCM ArrayBuffer to Float32Array
    private pcmToFloat32Array(buffer: ArrayBuffer, sampleRate: number = 16000): Float32Array {
        // Assuming 16-bit PCM
        const int16Array = new Int16Array(buffer);
        const float32Array = new Float32Array(int16Array.length);

        // Convert from 16-bit PCM to float32 (-1.0 to 1.0)
        for (let i = 0; i < int16Array.length; i++) {
            float32Array[i] = int16Array[i] / 32768.0;
        }

        return float32Array;
    }

    // Play PCM audio data
    async playPCMAudio(pcmData: ArrayBuffer, sampleRate: number = 16000): Promise<void> {
        try {
            await this.ensureAudioContext();

            if (!this.audioContext) {
                throw new Error('Audio context not available');
            }

            const audioData = this.pcmToFloat32Array(pcmData, sampleRate);
            const audioBuffer = this.audioContext.createBuffer(1, audioData.length, sampleRate);

            // Copy the PCM data to the audio buffer
            const channelData = audioBuffer.getChannelData(0);
            channelData.set(audioData);

            // Create source node and connect to destination
            const sourceNode = this.audioContext.createBufferSource();
            sourceNode.buffer = audioBuffer;
            sourceNode.connect(this.audioContext.destination);

            // Play the audio
            sourceNode.start(0);
            this.isPlaying = true;

            // Handle end of playback
            sourceNode.onended = () => {
                this.isPlaying = false;
            };

            this.sourceNode = sourceNode;

        } catch (error) {
            console.error('Failed to play PCM audio:', error);
            throw error;
        }
    }

    // Queue PCM audio chunks for continuous playback
    queuePCMAudio(pcmData: ArrayBuffer, sampleRate: number = 16000): void {
        const audioData = this.pcmToFloat32Array(pcmData, sampleRate);
        this.audioQueue.push(audioData);

        if (!this.isProcessing) {
            this.processAudioQueue(sampleRate);
        }
    }

    private async processAudioQueue(sampleRate: number): Promise<void> {
        if (this.isProcessing || this.audioQueue.length === 0) {
            return;
        }

        this.isProcessing = true;

        try {
            await this.ensureAudioContext();

            if (!this.audioContext) {
                throw new Error('Audio context not available');
            }

            while (this.audioQueue.length > 0) {
                const audioData = this.audioQueue.shift();
                if (!audioData) continue;

                const audioBuffer = this.audioContext.createBuffer(1, audioData.length, sampleRate);
                const channelData = audioBuffer.getChannelData(0);
                channelData.set(audioData);

                const sourceNode = this.audioContext.createBufferSource();
                sourceNode.buffer = audioBuffer;
                sourceNode.connect(this.audioContext.destination);

                // Wait for current audio to finish before playing next
                await new Promise<void>((resolve) => {
                    sourceNode.onended = () => resolve();
                    sourceNode.start(0);
                });
            }
        } catch (error) {
            console.error('Failed to process audio queue:', error);
        } finally {
            this.isProcessing = false;
        }
    }

    // Convert PCM to WAV format for download/traditional audio elements
    pcmToWav(pcmData: ArrayBuffer, sampleRate: number = 16000, numChannels: number = 1): Blob {
        const length = pcmData.byteLength;
        const buffer = new ArrayBuffer(44 + length);
        const view = new DataView(buffer);

        // WAV header
        const writeString = (offset: number, string: string) => {
            for (let i = 0; i < string.length; i++) {
                view.setUint8(offset + i, string.charCodeAt(i));
            }
        };

        writeString(0, 'RIFF');
        view.setUint32(4, 36 + length, true);
        writeString(8, 'WAVE');
        writeString(12, 'fmt ');
        view.setUint32(16, 16, true);
        view.setUint16(20, 1, true);
        view.setUint16(22, numChannels, true);
        view.setUint32(24, sampleRate, true);
        view.setUint32(28, sampleRate * numChannels * 2, true);
        view.setUint16(32, numChannels * 2, true);
        view.setUint16(34, 16, true);
        writeString(36, 'data');
        view.setUint32(40, length, true);

        // Copy PCM data
        const pcmView = new Uint8Array(pcmData);
        const wavView = new Uint8Array(buffer, 44);
        wavView.set(pcmView);

        return new Blob([buffer], { type: 'audio/wav' });
    }

    // Stop current playback
    stop(): void {
        if (this.sourceNode) {
            this.sourceNode.stop();
            this.sourceNode = null;
        }
        this.isPlaying = false;
        this.audioQueue = [];
        this.isProcessing = false;
    }

    // Check if audio is currently playing
    isCurrentlyPlaying(): boolean {
        return this.isPlaying;
    }

    // Get audio context state
    getAudioContextState(): string {
        return this.audioContext?.state || 'unavailable';
    }
}

// Create and export singleton instance
export const audioPlayerService = new AudioPlayerService();
export default audioPlayerService;
