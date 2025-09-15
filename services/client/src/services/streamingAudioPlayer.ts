// Real-time Streaming PCM Audio Player with Seamless Playback
export class StreamingPCMAudioPlayer {
    private audioContext: AudioContext | null = null;
    private isPlaying = false;
    private sampleRate = 22050;
    private channels = 1;
    private bufferedChunks: Float32Array[] = [];
    private scheduledSources: AudioBufferSourceNode[] = [];
    private nextScheduleTime = 0;
    private minBufferDuration = 0.05; // 50ms minimum buffer before starting
    private chunkDuration = 0.02; // 20ms chunks for ultra-smooth playback
    private isInitialized = false;
    private lastScheduledTime = 0;

    constructor() {
        this.initializeAudioContext();
    }

    private async initializeAudioContext() {
        try {
            this.audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();

            // Resume context if suspended
            if (this.audioContext.state === 'suspended') {
                await this.audioContext.resume();
            }

            // Set initial schedule time - start immediately when we have enough data
            this.nextScheduleTime = this.audioContext.currentTime;
            this.isInitialized = true;

            console.log('🎵 Streaming audio context initialized with seamless playback');
        } catch (error) {
            console.error('❌ Failed to initialize streaming audio context:', error);
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

    // Set audio format for streaming
    setAudioFormat(sampleRate: number, channels: number = 1) {
        this.sampleRate = sampleRate;
        this.channels = channels;
        console.log(`🎵 Streaming audio format set: ${sampleRate}Hz, ${channels} channels`);
    }

    // Add PCM chunk to streaming buffer
    async addPCMChunk(pcmData: ArrayBuffer) {
        try {
            await this.ensureAudioContext();

            if (!this.audioContext || !this.isInitialized) {
                console.error('❌ Audio context not ready for streaming');
                return;
            }

            // Convert PCM to Float32Array
            const audioData = this.pcmToFloat32Array(pcmData);

            // Add to buffer
            this.bufferedChunks.push(audioData);
            console.log(`📦 Added streaming chunk: ${audioData.length} samples (${pcmData.byteLength} bytes), total buffered: ${this.getTotalBufferedSamples()}`);

            // Try to schedule playback if we have enough data
            this.scheduleNextChunk();

        } catch (error) {
            console.error('❌ Error adding streaming PCM chunk:', error);
        }
    }

    private pcmToFloat32Array(buffer: ArrayBuffer): Float32Array {
        const int16Array = new Int16Array(buffer);
        const float32Array = new Float32Array(int16Array.length);

        // Convert from 16-bit PCM to float32 (-1.0 to 1.0)
        for (let i = 0; i < int16Array.length; i++) {
            float32Array[i] = int16Array[i] / 32768.0;
        }

        return float32Array;
    }

    private getTotalBufferedSamples(): number {
        return this.bufferedChunks.reduce((total, chunk) => total + chunk.length, 0);
    }

    private getTotalBufferedDuration(): number {
        return this.getTotalBufferedSamples() / this.sampleRate;
    }

    private scheduleNextChunk() {
        if (!this.audioContext || !this.isInitialized) {
            return;
        }

        // Check if we have enough data for at least one chunk
        const minSamples = Math.floor(this.sampleRate * this.chunkDuration);
        if (this.getTotalBufferedSamples() < minSamples) {
            return; // Not enough data yet
        }

        // If this is the first chunk and we haven't started playing yet,
        // check if we have enough buffer to start
        if (!this.isPlaying) {
            const minBufferSamples = Math.floor(this.sampleRate * this.minBufferDuration);
            if (this.getTotalBufferedSamples() < minBufferSamples) {
                return; // Wait for more buffer
            }

            // Start playing - set initial schedule time to current time
            this.isPlaying = true;
            this.nextScheduleTime = this.audioContext.currentTime + 0.01; // Start 10ms from now
            console.log('▶️ Starting seamless streaming audio playback');
        }

        // Take a chunk for playback
        const chunkData = this.takeChunkForPlayback(minSamples);

        if (chunkData.length > 0) {
            this.scheduleChunkPlayback(chunkData);
        }
    }

    private takeChunkForPlayback(targetSamples: number): Float32Array {
        let collectedSize = 0;
        const playbackChunks: Float32Array[] = [];

        // Take chunks until we have enough data
        while (collectedSize < targetSamples && this.bufferedChunks.length > 0) {
            const chunk = this.bufferedChunks[0];
            const remaining = targetSamples - collectedSize;

            if (chunk.length <= remaining) {
                // Take the whole chunk
                playbackChunks.push(this.bufferedChunks.shift()!);
                collectedSize += chunk.length;
            } else {
                // Take part of the chunk
                const partialChunk = chunk.slice(0, remaining);
                playbackChunks.push(partialChunk);

                // Update the remaining chunk
                this.bufferedChunks[0] = chunk.slice(remaining);
                collectedSize += remaining;
            }
        }

        // Concatenate all playback chunks
        if (playbackChunks.length === 0) {
            return new Float32Array(0);
        }

        const totalLength = playbackChunks.reduce((sum, chunk) => sum + chunk.length, 0);
        const result = new Float32Array(totalLength);
        let offset = 0;

        for (const chunk of playbackChunks) {
            result.set(chunk, offset);
            offset += chunk.length;
        }

        return result;
    }

    private scheduleChunkPlayback(audioData: Float32Array) {
        if (!this.audioContext) {
            return;
        }

        // Create audio buffer
        const audioBuffer = this.audioContext.createBuffer(
            this.channels,
            audioData.length,
            this.sampleRate
        );

        // Copy audio data
        const channelData = audioBuffer.getChannelData(0);
        channelData.set(audioData);

        // Create source node
        const source = this.audioContext.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(this.audioContext.destination);

        // Schedule playback at the calculated time
        const startTime = this.nextScheduleTime;
        source.start(startTime);

        // Update next schedule time to start immediately after this chunk ends
        // This ensures seamless, gapless playback
        this.nextScheduleTime = startTime + audioBuffer.duration;

        // Track the source for cleanup
        this.scheduledSources.push(source);

        // Clean up ended sources
        source.onended = () => {
            const index = this.scheduledSources.indexOf(source);
            if (index > -1) {
                this.scheduledSources.splice(index, 1);
            }
        };

        console.log(`▶️ Scheduled seamless chunk: ${audioBuffer.duration.toFixed(3)}s at ${startTime.toFixed(3)}s, next at ${this.nextScheduleTime.toFixed(3)}s`);

        // Schedule next chunk if we have more data
        if (this.getTotalBufferedSamples() > 0) {
            // Use setTimeout to avoid blocking the main thread
            setTimeout(() => this.scheduleNextChunk(), 0);
        }
    }

    // Stop streaming playback
    stop() {
        // Stop all scheduled sources
        this.scheduledSources.forEach(source => {
            try {
                source.stop();
            } catch (e) {
                // Already stopped
            }
        });
        this.scheduledSources = [];

        this.isPlaying = false;
        this.bufferedChunks = [];
        this.nextScheduleTime = this.audioContext ? this.audioContext.currentTime : 0;
        this.lastScheduledTime = 0;

        console.log('🛑 Streaming audio stopped');
    }

    // Check if currently playing
    isCurrentlyPlaying(): boolean {
        return this.isPlaying;
    }

    // Get current buffer size in samples
    getBufferSize(): number {
        return this.getTotalBufferedSamples();
    }

    // Get buffer health (0-1, where 1 is fully buffered)
    getBufferHealth(): number {
        const minBufferSamples = Math.floor(this.sampleRate * this.minBufferDuration);
        return Math.min(1.0, this.getTotalBufferedSamples() / minBufferSamples);
    }

    // Get buffer duration in seconds
    getBufferDuration(): number {
        return this.getTotalBufferedDuration();
    }

    // Cleanup
    cleanup() {
        this.stop();

        if (this.audioContext) {
            try {
                this.audioContext.close();
            } catch (e) {
                // Ignore
            }
            this.audioContext = null;
        }
    }
}

// Export singleton instance
export const streamingPCMAudioPlayer = new StreamingPCMAudioPlayer();