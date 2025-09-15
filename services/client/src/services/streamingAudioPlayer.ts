// Real-time Streaming PCM Audio Player with Seamless Playback
import { getAudioContext } from './audio';

export class StreamingPCMAudioPlayer {
    private audioContext: AudioContext | null = null;
    private externalAudioContext: AudioContext | null = null;
    private isPlaying = false;
    private sampleRate = 22050;
    private channels = 1;
    private bufferedChunks: Float32Array[] = [];
    private scheduledSources: AudioBufferSourceNode[] = [];
    private nextScheduleTime = 0;
    private minBufferDuration = 0.025; // 25ms minimum buffer before starting
    private chunkDuration = 0.010; // 10ms chunks for smoother playback
    private isInitialized = false;
    private lastScheduledTime = 0;

    constructor(externalAudioContext?: AudioContext) {
        this.externalAudioContext = externalAudioContext || null;
        this.initializeAudioContext();
    }

    private async initializeAudioContext() {
        try {
            // Use external context if provided, otherwise create new one
            if (this.externalAudioContext) {
                this.audioContext = this.externalAudioContext;
            } else {
                this.audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();
            }

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
            console.log(`🎵 addPCMChunk called with ${pcmData.byteLength} bytes`);
            await this.ensureAudioContext();

            if (!this.audioContext || !this.isInitialized) {
                console.error('❌ Audio context not ready for streaming - context:', !!this.audioContext, 'initialized:', this.isInitialized);
                return;
            }

            console.log(`🎵 Audio context ready, processing chunk`);
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

            // Start playing - set initial schedule time with buffer for larger chunks
            this.isPlaying = true;
            this.nextScheduleTime = this.audioContext.currentTime + 0.015; // Start 15ms from now
            console.log('▶️ Starting smooth streaming audio playback');
        }

        // Schedule multiple chunks ahead for smoother playback
        this.scheduleAheadChunks();
    }

    private scheduleAheadChunks() {
        // Very conservative - only schedule 1 chunk ahead to avoid conflicts
        const maxChunksAhead = 1; // Only one chunk at a time

        for (let i = 0; i < maxChunksAhead; i++) {
            const minSamples = Math.floor(this.sampleRate * this.chunkDuration);
            if (this.getTotalBufferedSamples() < minSamples) {
                break; // Not enough data for more chunks
            }

            // Take a chunk for playback
            const chunkData = this.takeChunkForPlayback(minSamples);

            if (chunkData.length > 0) {
                this.scheduleChunkPlayback(chunkData);
            } else {
                break; // No more data
            }
        }

        // Don't schedule more aggressively - let natural flow handle it
    }

    private takeChunkForPlayback(targetSamples: number): Float32Array {
        let collectedSize = 0;
        const playbackChunks: Float32Array[] = [];

        // Take chunks until we have enough data - simplified, no overlap
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
            console.error('❌ No audio context available');
            return;
        }

        // Ensure audio context is running
        if (this.audioContext.state !== 'running') {
            console.warn(`⚠️ Audio context state: ${this.audioContext.state}, attempting to resume`);
            this.audioContext.resume().catch(error => {
                console.error('❌ Failed to resume audio context:', error);
                return;
            });
        }

        // Skip crossfade for now - test if basic chunking works
        // const smoothedAudioData = this.applyCrossfade(audioData);
        const smoothedAudioData = audioData; // No crossfade

        // Create audio buffer
        const audioBuffer = this.audioContext.createBuffer(
            this.channels,
            smoothedAudioData.length,
            this.sampleRate
        );

        // Copy audio data
        const channelData = audioBuffer.getChannelData(0);
        channelData.set(smoothedAudioData);

        // Create source node
        const source = this.audioContext.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(this.audioContext.destination);

        // Schedule playback at the calculated time
        const startTime = this.nextScheduleTime;
        const currentTime = this.audioContext.currentTime;

        // Conservative timing for larger chunks - add buffer to prevent conflicts
        let actualStartTime = startTime;
        if (startTime <= currentTime + 0.005) { // Add 5ms buffer for larger chunks
            // We're close to schedule time - add offset to prevent conflicts
            actualStartTime = currentTime + 0.008; // 8ms offset
            if (startTime < currentTime) {
                console.log(`⚡ Schedule adjustment (${(currentTime - startTime).toFixed(3)}s behind), using ${actualStartTime.toFixed(3)}s`);
            }
        }

        // Start at the precise scheduled time for seamless playback
        try {
            source.start(actualStartTime);
            console.log(`▶️ Source started successfully at ${actualStartTime.toFixed(3)}s`);
        } catch (error) {
            console.error(`❌ Failed to start source at ${actualStartTime.toFixed(3)}s:`, error);
            return;
        }

        // Update next schedule time for seamless playback
        // No overlap - just butt chunks together precisely
        this.nextScheduleTime = actualStartTime + audioBuffer.duration;

        // Track the source for cleanup
        this.scheduledSources.push(source);

        // Clean up ended sources
        source.onended = () => {
            const index = this.scheduledSources.indexOf(source);
            if (index > -1) {
                this.scheduledSources.splice(index, 1);
            }
            console.log(`✅ Source ended for chunk at ${actualStartTime.toFixed(3)}s`);
        };

        console.log(`▶️ Scheduled chunk: ${audioBuffer.duration.toFixed(3)}s at ${actualStartTime.toFixed(3)}s, current time: ${currentTime.toFixed(3)}s`);

        // Schedule next chunk with appropriate delay for larger chunks
        if (this.getTotalBufferedSamples() > 0) {
            // Longer delay for larger chunks to prevent overwhelming
            setTimeout(() => this.scheduleNextChunk(), 3);
        }
    }

    // Stop streaming playback
    stop() {
        console.log('🛑 Streaming audio stopped - clearing buffers');
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
        const optimalBufferSamples = Math.floor(this.sampleRate * 0.08); // 80ms optimal buffer for larger chunks
        const currentSamples = this.getTotalBufferedSamples();

        if (currentSamples >= optimalBufferSamples) {
            return 1.0; // Fully buffered
        } else if (currentSamples >= minBufferSamples) {
            // Scale between min and optimal
            return (currentSamples - minBufferSamples) / (optimalBufferSamples - minBufferSamples);
        } else {
            return 0.0; // Below minimum
        }
    }

    // Get detailed buffer status
    getBufferStatus(): { health: number; duration: number; samples: number; isLow: boolean } {
        const samples = this.getTotalBufferedSamples();
        const duration = this.getTotalBufferedDuration();
        const health = this.getBufferHealth();
        const minBufferSamples = Math.floor(this.sampleRate * this.minBufferDuration);

        return {
            health,
            duration,
            samples,
            isLow: samples < minBufferSamples
        };
    }

    // Get performance metrics for debugging
    getPerformanceMetrics(): {
        bufferHealth: number;
        bufferedDuration: number;
        isPlaying: boolean;
        scheduledSources: number;
        nextScheduleTime: number;
        currentTime: number;
        scheduleLag: number;
    } {
        const currentTime = this.audioContext?.currentTime || 0;
        return {
            bufferHealth: this.getBufferHealth(),
            bufferedDuration: this.getTotalBufferedDuration(),
            isPlaying: this.isPlaying,
            scheduledSources: this.scheduledSources.length,
            nextScheduleTime: this.nextScheduleTime,
            currentTime,
            scheduleLag: this.nextScheduleTime - currentTime
        };
    }

    // Pre-warm audio context for better performance
    async prewarm() {
        try {
            await this.ensureAudioContext();
            if (this.audioContext) {
                // Create and immediately discard a larger silent buffer to warm up the context
                const silentBuffer = this.audioContext.createBuffer(1, Math.floor(this.sampleRate * 0.01), 22050); // 10ms buffer
                const source = this.audioContext.createBufferSource();
                source.buffer = silentBuffer;
                source.connect(this.audioContext.destination);
                source.start();
                source.stop(this.audioContext.currentTime + 0.001);
                console.log('🔥 Audio context pre-warmed for optimal performance');
            }
        } catch (error) {
            console.warn('⚠️ Failed to pre-warm audio context:', error);
        }
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

// Export singleton instance - will use audio service context when available
export const streamingPCMAudioPlayer = new StreamingPCMAudioPlayer(getAudioContext() || undefined);