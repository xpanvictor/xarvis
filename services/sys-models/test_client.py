#!/usr/bin/env python3
"""
Test client for Silero VAD Service

Simple script to test the VAD service with sample audio or generated audio.
"""

import requests
import numpy as np
import io
import wave
import time
import json
from typing import Optional

def generate_test_audio(duration_seconds: float = 2.0, sample_rate: int = 16000, 
                       frequency: float = 440.0, with_silence: bool = True) -> bytes:
    """Generate test audio with optional silence periods"""
    
    # Generate sine wave for speech simulation
    t = np.linspace(0, duration_seconds, int(sample_rate * duration_seconds))
    
    if with_silence:
        # Create pattern: silence - speech - silence
        speech_start = int(0.3 * len(t))
        speech_end = int(0.7 * len(t))
        
        audio = np.zeros_like(t)
        audio[speech_start:speech_end] = 0.3 * np.sin(2 * np.pi * frequency * t[speech_start:speech_end])
        
        # Add some noise to make it more realistic
        noise = np.random.normal(0, 0.01, len(audio))
        audio += noise
    else:
        # Pure sine wave (simulates continuous speech)
        audio = 0.3 * np.sin(2 * np.pi * frequency * t)
    
    # Convert to 16-bit PCM
    audio_int16 = (audio * 32767).astype(np.int16)
    
    # Create WAV file in memory
    buffer = io.BytesIO()
    with wave.open(buffer, 'wb') as wav_file:
        wav_file.setnchannels(1)  # Mono
        wav_file.setsampwidth(2)  # 16-bit
        wav_file.setframerate(sample_rate)
        wav_file.writeframes(audio_int16.tobytes())
    
    return buffer.getvalue()

def test_vad_service(base_url: str = "http://localhost:8001"):
    """Test the VAD service with various scenarios"""
    
    print(f"Testing System Models Service (VAD) at {base_url}")
    print("=" * 50)
    
    # Test 1: Health check
    print("1. Testing health endpoint...")
    try:
        response = requests.get(f"{base_url}/health", timeout=5)
        if response.status_code == 200:
            health_data = response.json()
            print(f"   ✅ Health check passed: {health_data}")
        else:
            print(f"   ❌ Health check failed: {response.status_code}")
            return
    except Exception as e:
        print(f"   ❌ Health check error: {e}")
        return
    
    # Test 2: Audio with speech
    print("\n2. Testing audio with speech...")
    try:
        audio_with_speech = generate_test_audio(duration_seconds=2.0, with_silence=True)
        
        start_time = time.time()
        response = requests.post(
            f"{base_url}/vad",
            files={"file": ("test_audio.wav", audio_with_speech, "audio/wav")},
            timeout=10
        )
        request_time = time.time() - start_time
        
        if response.status_code == 200:
            result = response.json()
            print(f"   ✅ Request completed in {request_time*1000:.1f}ms")
            print(f"   Has voice: {result['has_voice']}")
            print(f"   Confidence: {result['confidence']:.3f}")
            print(f"   Segments: {len(result['segments'])}")
            print(f"   Processing time: {result['processing_time_ms']:.1f}ms")
            print(f"   Audio duration: {result['audio_duration_ms']:.1f}ms")
            
            for i, segment in enumerate(result['segments']):
                print(f"      Segment {i+1}: {segment['start']:.2f}s - {segment['end']:.2f}s (conf: {segment['confidence']:.3f})")
        else:
            print(f"   ❌ Request failed: {response.status_code} - {response.text}")
    except Exception as e:
        print(f"   ❌ Request error: {e}")
    
    # Test 3: Audio without speech (silence)
    print("\n3. Testing silent audio...")
    try:
        silent_audio = generate_test_audio(duration_seconds=1.0, frequency=0, with_silence=False)
        # Make it very quiet
        silent_audio = silent_audio[:100] + b'\x00' * (len(silent_audio) - 100)
        
        response = requests.post(
            f"{base_url}/vad",
            files={"file": ("silent_audio.wav", silent_audio, "audio/wav")},
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            print(f"   ✅ Silent audio processed")
            print(f"   Has voice: {result['has_voice']} (should be False)")
            print(f"   Confidence: {result['confidence']:.3f}")
            print(f"   Segments: {len(result['segments'])} (should be 0)")
        else:
            print(f"   ❌ Request failed: {response.status_code}")
    except Exception as e:
        print(f"   ❌ Request error: {e}")
    
    # Test 4: Custom parameters
    print("\n4. Testing custom parameters...")
    try:
        audio_data = generate_test_audio(duration_seconds=3.0, with_silence=True)
        
        params = {
            "threshold": 0.3,
            "min_speech_duration_ms": 100,
            "min_silence_duration_ms": 50
        }
        
        response = requests.post(
            f"{base_url}/vad",
            files={"file": ("test_audio.wav", audio_data, "audio/wav")},
            params=params,
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            print(f"   ✅ Custom parameters processed")
            print(f"   Has voice: {result['has_voice']}")
            print(f"   Confidence: {result['confidence']:.3f}")
            print(f"   Segments: {len(result['segments'])}")
        else:
            print(f"   ❌ Request failed: {response.status_code}")
    except Exception as e:
        print(f"   ❌ Request error: {e}")
    
    # Test 5: Upload endpoint
    print("\n5. Testing upload endpoint...")
    try:
        audio_data = generate_test_audio(duration_seconds=1.5, with_silence=True)
        
        response = requests.post(
            f"{base_url}/vad/stream",
            files={"file": ("upload_test.wav", audio_data, "audio/wav")},
            timeout=10
        )
        
        if response.status_code == 200:
            result = response.json()
            print(f"   ✅ Upload endpoint works")
            print(f"   Has voice: {result['has_voice']}")
            print(f"   Confidence: {result['confidence']:.3f}")
        else:
            print(f"   ❌ Upload endpoint failed: {response.status_code}")
    except Exception as e:
        print(f"   ❌ Upload endpoint error: {e}")
    
    print("\n" + "=" * 50)
    print("Test completed!")

def main():
    import argparse
    
    parser = argparse.ArgumentParser(description="Test System Models Service")
    parser.add_argument("--url", default="http://localhost:8001", 
                       help="Base URL of the System Models service")
    
    args = parser.parse_args()
    
    test_vad_service(args.url)

if __name__ == "__main__":
    main()
