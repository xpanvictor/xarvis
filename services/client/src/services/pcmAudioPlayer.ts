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
    private htmlAudio: HTMLAudioElement | null = null;
    private htmlAudioURL: string | null = null;
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
    private playbackMode: 'wav' | 'webaudio' = 'wav';

    // Public API ---------------------------------------------------------------
    // Call at the start of a new server audio stream
    startNewStream(stopCurrent: boolean = true) {
        console.log('🔁 Starting new audio stream; resetting collection');
        if (stopCurrent) {
            this.stopAllAudio();
        } else {
            // If not stopping, at least clear buffer for next concatenation
            this.isCollecting = false;
            this.collectedPCMData = [];
        }
    }

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

            // Do not trim bytes; preserve exact stream bytes. If total length
            // ends up odd, the final decode will ignore the last byte safely.

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

        // If currently playing, we'll replace the source safely when starting
        // the new buffer (no need to clear collected data here).

        try {
            console.log(`🎵 Playing collected audio: ${this.collectedPCMData.length} chunks`);

            // Initialize audio context if using WebAudio mode
            if (this.playbackMode === 'webaudio') {
                await this.ensureContext();
                console.log('✅ Audio context ensured');
            }

            // Concatenate all PCM data
            const concatenatedPCM = this.concatenateAllPCMData();
            if (!concatenatedPCM) {
                console.error('❌ Failed to concatenate PCM data');
                return;
            }
            console.log(`✅ Concatenated PCM data: ${concatenatedPCM.byteLength} bytes`);

            if (this.playbackMode === 'webaudio') {
                // Convert to AudioBuffer and play via WebAudio
                const audioBuffer = this.pcmToAudioBuffer(concatenatedPCM);
                if (!audioBuffer) {
                    console.error('❌ Failed to create AudioBuffer');
                    return;
                }
                console.log(`✅ Created AudioBuffer: ${audioBuffer.duration.toFixed(3)}s duration`);
                await this.playAudioBuffer(audioBuffer);
            } else {
                // Build a WAV blob and play via HTMLAudioElement (matches backend debug file)
                const wav = this.buildWavBlob(concatenatedPCM);
                try {
                    await this.playWavPreferMediaElement(wav);
                } catch (e) {
                    console.warn('⚠️ HTMLAudio playback failed, falling back to WebAudio decode', e);
                    await this.playWavViaWebAudio(wav);
                }
            }

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

        // Stop and release HTMLAudio element
        if (this.htmlAudio) {
            try { this.htmlAudio.pause(); } catch {}
            this.htmlAudio.src = '';
            this.htmlAudio = null;
        }
        if (this.htmlAudioURL) {
            try { URL.revokeObjectURL(this.htmlAudioURL); } catch {}
            this.htmlAudioURL = null;
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

    setPlaybackMode(mode: 'wav' | 'webaudio') {
        this.playbackMode = mode;
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

        // Stop any existing source before starting a new one
        if (this.currentSource) {
            try { this.currentSource.stop(); } catch {}
            this.currentSource.disconnect();
            this.currentSource = null;
        }

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

    private buildWavBlob(pcmData: ArrayBuffer): Blob {
        const { sampleRate, channels, bitsPerSample } = this.format;
        const bytesPerSample = bitsPerSample / 8;
        const dataLength = pcmData.byteLength;
        const blockAlign = channels * bytesPerSample;
        const byteRate = sampleRate * blockAlign;

        const header = new ArrayBuffer(44);
        const view = new DataView(header);
        const writeString = (offset: number, s: string) => {
            for (let i = 0; i < s.length; i++) view.setUint8(offset + i, s.charCodeAt(i));
        };

        // RIFF chunk descriptor
        writeString(0, 'RIFF');
        view.setUint32(4, 36 + dataLength, true); // ChunkSize
        writeString(8, 'WAVE');

        // fmt sub-chunk
        writeString(12, 'fmt ');
        view.setUint32(16, 16, true); // Subchunk1Size for PCM
        view.setUint16(20, 1, true);  // AudioFormat = 1 (PCM)
        view.setUint16(22, channels, true);
        view.setUint32(24, sampleRate, true);
        view.setUint32(28, byteRate, true);
        view.setUint16(32, blockAlign, true);
        view.setUint16(34, bitsPerSample, true);

        // data sub-chunk
        writeString(36, 'data');
        view.setUint32(40, dataLength, true);

        const wavBuffer = new Uint8Array(44 + dataLength);
        wavBuffer.set(new Uint8Array(header), 0);
        wavBuffer.set(new Uint8Array(pcmData), 44);
        return new Blob([wavBuffer], { type: 'audio/wav' });
    }

    private playWavPreferMediaElement(wav: Blob): Promise<void> {
        return new Promise<void>((resolve, reject) => {
            try {
                // Tear down previous element/url
                if (this.htmlAudio) {
                    try { this.htmlAudio.pause(); } catch {}
                    this.htmlAudio.src = '';
                    this.htmlAudio = null;
                }
                if (this.htmlAudioURL) {
                    try { URL.revokeObjectURL(this.htmlAudioURL); } catch {}
                    this.htmlAudioURL = null;
                }

                const url = URL.createObjectURL(wav);
                this.htmlAudioURL = url;
                const audio = new Audio();
                audio.preload = 'auto';
                audio.src = url;

                audio.onerror = (ev) => {
                    this.isPlaying = false;
                    reject(new Error('HTMLAudio error'));
                };
                audio.onended = () => {
                    this.isPlaying = false;
                    try {
                        if (this.htmlAudioURL) URL.revokeObjectURL(this.htmlAudioURL);
                    } catch {}
                    this.htmlAudioURL = null;
                    resolve();
                };
                audio.oncanplaythrough = () => {
                    // Try to play once data is buffered
                    audio.play().then(() => {
                        this.isPlaying = true;
                        resolve();
                    }).catch((e) => {
                        this.isPlaying = false;
                        reject(e);
                    });
                };
                // Kick load
                audio.load();
                this.htmlAudio = audio;
            } catch (e) {
                reject(e as any);
            }
        });
    }

    private async playWavViaWebAudio(wav: Blob): Promise<void> {
        const ctx = await this.ensureContext();
        const ab = await wav.arrayBuffer();
        const audioBuffer = await new Promise<AudioBuffer>((resolve, reject) => {
            ctx.decodeAudioData(ab, resolve, reject);
        });
        await this.playAudioBuffer(audioBuffer);
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
            const samples = new Int16Array(pcmData);

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
                        const s = samples[sampleIndex];
                        let sample = s / 32768.0;

                        // No harsh limiting - keep it natural
                        if (sample > 1.0) sample = 1.0;
                        else if (sample < -1.0) sample = -1.0;

                        channelData[i] = sample;
                    } else {
                        channelData[i] = 0;
                    }
                }

                // No fades; play exactly as concatenated, like backend
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
