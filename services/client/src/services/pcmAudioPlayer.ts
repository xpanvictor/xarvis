// Simple PCM Audio Player - Collect All Then Play (Like Backend)

export interface AudioChunk {
    index: number;
    data: ArrayBuffer;
    received: boolean;
}

export interface AudioFormat {
    format: 'pcm';
    sampleRate: number;
    channels: 1 | 2;
    bitsPerSample: 16;
    encoding: 's16le';
}

class PCMAudioPlayerService {
    private audioContext: AudioContext | null = null;
    private format: AudioFormat = {
        format: 'pcm',
        sampleRate: 22050,
        channels: 1,
        bitsPerSample: 16,
        encoding: 's16le',
    };

    // Simple collection approach - like backend
    private collectedPCMData: ArrayBuffer[] = [];
    private isCollecting = false;
    private currentSource: AudioBufferSourceNode | null = null;
    private isPlaying = false;

    // Public API ---------------------------------------------------------------

    setAudioFormat(format: AudioFormat) {
        if (format.format !== 'pcm' || format.encoding !== 's16le') {
            throw new Error(`Only s16le PCM supported, got ${format.format}/${format.encoding}`);
        }
        if (format.bitsPerSample !== 16) {
            throw new Error(`Only 16-bit PCM supported, got ${format.bitsPerSample}`);
        }
        if (format.channels !== 1 && format.channels !== 2) {
            throw new Error(`Unsupported channel count: ${format.channels}`);
        }

        this.format = { ...format };
        console.log('🎵 Audio format set:', format);
    }

    async addPCMChunk(pcmData: ArrayBuffer) {
        try {
            if (!pcmData || pcmData.byteLength === 0) {
                console.warn('⚠️ Empty PCM chunk received, skipping');
                return;
            }

            // Ensure 16-bit alignment (drop last byte if odd-sized)
            if (pcmData.byteLength % 2 !== 0) {
                console.warn(`⚠️ PCM chunk has odd length (${pcmData.byteLength}); trimming last byte for 16-bit alignment`);
                const trimmed = pcmData.slice(0, pcmData.byteLength - 1);
                pcmData = trimmed;
            }

            // Start collecting if not already
            if (!this.isCollecting) {
                console.log('📦 Started collecting PCM chunks');
                this.isCollecting = true;
                this.collectedPCMData = [];
            }

            // Add chunk to collection
            this.collectedPCMData.push(pcmData.slice(0)); // Make a copy
            console.log(`📦 Collected chunk ${this.collectedPCMData.length} (${pcmData.byteLength} bytes)`);

        } catch (error) {
            console.error('❌ Error adding PCM chunk:', error);
        }
    }

    // Call this when all audio chunks are received (e.g., stream completed)
    async playCollectedAudio() {
        console.log(`🎵 playCollectedAudio called with ${this.collectedPCMData.length} chunks`);

        if (this.collectedPCMData.length === 0) {
            console.warn('⚠️ No PCM data collected to play');
            return;
        }

        try {
            console.log(`🎵 Playing collected audio: ${this.collectedPCMData.length} chunks`);

            // Initialize audio context
            await this.ensureContext();
            console.log('✅ Audio context ensured');

            // Concatenate all PCM data
            const concatenatedPCM = this.concatenateAllPCMData();
            if (!concatenatedPCM) {
                console.error('❌ Failed to concatenate PCM data');
                return;
            }
            console.log(`✅ Concatenated PCM data: ${concatenatedPCM.byteLength} bytes`);

            // Convert to AudioBuffer
            const audioBuffer = this.pcmToAudioBuffer(concatenatedPCM);
            if (!audioBuffer) {
                console.error('❌ Failed to create AudioBuffer');
                return;
            }
            console.log(`✅ Created AudioBuffer: ${audioBuffer.duration.toFixed(3)}s duration`);

            // Play the complete audio
            await this.playAudioBuffer(audioBuffer);

        } catch (error) {
            console.error('❌ Error playing collected audio:', error);
        } finally {
            // Reset collection state
            this.isCollecting = false;
            this.collectedPCMData = [];
        }
    }

    stopAllAudio() {
        console.log('🛑 Stopping all audio');

        // Stop current source
        if (this.currentSource) {
            try {
                this.currentSource.stop();
            } catch (e) {
                // Already stopped
            }
            this.currentSource = null;
        }

        // Reset state
        this.isPlaying = false;
        this.isCollecting = false;
        this.collectedPCMData = [];

        // Close audio context for clean reset
        if (this.audioContext) {
            try {
                this.audioContext.close();
            } catch (e) {
                // Ignore
            }
            this.audioContext = null;
        }
    }

    getQueueLength() {
        return this.collectedPCMData.length;
    }

    getIsPlaying() {
        return this.isPlaying;
    }

    cleanup() {
        this.stopAllAudio();
    }

    // Private Implementation --------------------------------------------------

    private concatenateAllPCMData(): ArrayBuffer | null {
        if (this.collectedPCMData.length === 0) return null;

        try {
            // Calculate total size
            let totalBytes = 0;
            for (const chunk of this.collectedPCMData) {
                totalBytes += chunk.byteLength;
            }

            console.log(`🔧 Concatenating ${this.collectedPCMData.length} chunks, total: ${totalBytes} bytes`);

            // Create combined buffer
            const combined = new ArrayBuffer(totalBytes);
            const combinedView = new Uint8Array(combined);
            let offset = 0;

            // Copy all chunks
            for (const chunk of this.collectedPCMData) {
                const chunkView = new Uint8Array(chunk);
                combinedView.set(chunkView, offset);
                offset += chunk.byteLength;
            }

            return combined;

        } catch (error) {
            console.error('❌ Error concatenating PCM data:', error);
            return null;
        }
    }

    private async playAudioBuffer(audioBuffer: AudioBuffer) {
        if (!this.audioContext) {
            console.error('❌ No audio context available');
            return;
        }

        console.log(`▶️ Playing complete audio: ${audioBuffer.duration.toFixed(2)}s duration`);
        console.log(`📊 Audio details: ${audioBuffer.numberOfChannels} channels, ${audioBuffer.sampleRate}Hz, ${audioBuffer.length} frames`);

        // Create source
        const source = this.audioContext.createBufferSource();
        source.buffer = audioBuffer;
        source.connect(this.audioContext.destination);

        // Track current source
        this.currentSource = source;
        this.isPlaying = true;

        // Setup completion handler
        source.onended = () => {
            console.log('🏁 Audio playback completed');
            this.isPlaying = false;
            this.currentSource = null;
        };

        // Start playback immediately
        source.start();
        console.log('✅ Audio playback started');
    }

    private async ensureContext(): Promise<AudioContext> {
        if (!this.audioContext) {
            this.audioContext = new (window.AudioContext || (window as any).webkitAudioContext)();
        }

        if (this.audioContext.state === 'suspended') {
            await this.audioContext.resume();
        }

        return this.audioContext;
    }

    private pcmToAudioBuffer(pcmData: ArrayBuffer): AudioBuffer | null {
        if (!this.audioContext) return null;

        try {
            const format = this.format;
            const bytesPerSample = format.bitsPerSample / 8;
            const totalSamples = pcmData.byteLength / bytesPerSample;
            const frames = Math.floor(totalSamples / format.channels);

            if (frames <= 0) {
                console.warn('⚠️ No audio frames to process');
                return null;
            }

            console.log(`🔧 Converting PCM to AudioBuffer: ${frames} frames, ${format.channels} channels`);

            // Create audio buffer
            const buffer = this.audioContext.createBuffer(
                format.channels,
                frames,
                format.sampleRate
            );

            // Convert PCM data (little-endian 16-bit signed)
            const samples = new Int16Array(pcmData.byteLength / 2);
            // Typed arrays read in platform endianness; use DataView to be explicit LE
            const dv = new DataView(pcmData);
            for (let i = 0; i < samples.length; i++) {
                samples[i] = dv.getInt16(i * 2, true);
            }

            // Optional: compute DC offset and remove it (helps with low hum)
            let mean = 0;
            const nForMean = Math.min(samples.length, 48000); // sample a subset for speed
            for (let i = 0; i < nForMean; i++) mean += samples[i];
            mean /= nForMean || 1;

            for (let channel = 0; channel < format.channels; channel++) {
                const channelData = buffer.getChannelData(channel);

                for (let i = 0; i < frames; i++) {
                    let sampleIndex: number;

                    if (format.channels === 1) {
                        // Mono audio
                        sampleIndex = i;
                    } else {
                        // Stereo audio (interleaved)
                        sampleIndex = i * format.channels + channel;
                    }

                    if (sampleIndex < samples.length) {
                        // Convert 16-bit signed integer to float [-1, 1]
                        let s = samples[sampleIndex] - mean; // remove DC offset
                        let sample = s >= 0 ? s / 32767.0 : s / 32768.0;

                        // No harsh limiting - keep it natural
                        if (sample > 1.0) sample = 1.0;
                        else if (sample < -1.0) sample = -1.0;

                        channelData[i] = sample;
                    } else {
                        channelData[i] = 0;
                    }
                }

                // Apply gentle 5ms fade-in/out to reduce boundary clicks
                const fadeSamples = Math.min(Math.floor(format.sampleRate * 0.005), frames);
                for (let i = 0; i < fadeSamples; i++) {
                    const gainIn = i / fadeSamples;
                    channelData[i] *= gainIn;
                    const j = frames - 1 - i;
                    const gainOut = i / fadeSamples;
                    channelData[j] *= gainOut;
                }
            }

            console.log(`✅ AudioBuffer created: ${buffer.duration.toFixed(3)}s`);
            return buffer;

        } catch (error) {
            console.error('❌ Error converting PCM to AudioBuffer:', error);
            return null;
        }
    }
}

// Export singleton
export const pcmAudioPlayer = new PCMAudioPlayerService();
export default pcmAudioPlayer;
