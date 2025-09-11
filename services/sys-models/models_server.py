#!/usr/bin/env python3
"""
System Models Service

A FastAPI-based microservice that provides multiple lightweight ML models including:
- Voice Activity Detection (VAD) using Silero VAD
- Additional system-level ML models (expandable)

Exposes REST API endpoints for real-time model inference.
"""

import io
import logging
import time
from typing import List, Dict, Any, Optional

import torch
import torchaudio
import numpy as np
from fastapi import FastAPI, File, HTTPException, UploadFile
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
import uvicorn

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Pydantic models for request/response
class VADConfig(BaseModel):
    """Configuration for VAD processing"""
    threshold: float = Field(default=0.5, ge=0.0, le=1.0, description="Voice detection threshold")
    min_speech_duration_ms: int = Field(default=250, ge=1, description="Minimum speech duration in milliseconds")
    min_silence_duration_ms: int = Field(default=100, ge=1, description="Minimum silence duration in milliseconds")
    window_size_samples: int = Field(default=1536, ge=512, description="Window size for processing in samples")
    sampling_rate: int = Field(default=16000, description="Expected sampling rate")

class VADSegment(BaseModel):
    """A voice activity segment"""
    start: float = Field(description="Start time in seconds")
    end: float = Field(description="End time in seconds")
    confidence: float = Field(ge=0.0, le=1.0, description="Confidence score")

class VADResponse(BaseModel):
    """Response from VAD analysis"""
    has_voice: bool = Field(description="Whether voice activity was detected")
    confidence: float = Field(ge=0.0, le=1.0, description="Overall confidence score")
    segments: List[VADSegment] = Field(description="List of voice activity segments")
    processing_time_ms: float = Field(description="Processing time in milliseconds")
    audio_duration_ms: float = Field(description="Duration of processed audio in milliseconds")

class HealthResponse(BaseModel):
    """Health check response"""
    status: str
    model_loaded: bool
    version: str

# Global variables for model
model = None
utils = None
get_speech_timestamps = None
read_audio = None

def load_silero_model():
    """Load the Silero VAD model"""
    global model, utils, get_speech_timestamps, read_audio
    
    try:
        logger.info("Loading Silero VAD model...")
        start_time = time.time()
        
        # Load Silero VAD model
        model, utils = torch.hub.load(
            repo_or_dir='snakers4/silero-vad',
            model='silero_vad',
            force_reload=False,
            onnx=False
        )
        
        # Extract utility functions
        (get_speech_timestamps, _, read_audio, _, _) = utils
        
        load_time = time.time() - start_time
        logger.info(f"Silero VAD model loaded successfully in {load_time:.2f}s")
        
        return True
        
    except Exception as e:
        logger.error(f"Failed to load Silero VAD model: {e}")
        return False

def process_audio_bytes(audio_bytes: bytes, config: VADConfig) -> VADResponse:
    """Process audio bytes and return VAD results"""
    global model, get_speech_timestamps, read_audio
    
    if model is None:
        raise HTTPException(status_code=503, detail="VAD model not loaded")
    
    start_time = time.time()
    logger.info("sampl rate:", config.sampling_rate)
    
    try:
        # Convert bytes to audio tensor
        audio_io = io.BytesIO(audio_bytes)
        
        # Read audio using Silero's utility function
        # This handles various audio formats and converts to the expected format
        wav = read_audio(audio_io, sampling_rate=config.sampling_rate)
        
        # Calculate audio duration
        audio_duration_ms = len(wav) / config.sampling_rate * 1000
        
        # Get speech timestamps using Silero VAD
        speech_timestamps = get_speech_timestamps(
            wav,
            model,
            sampling_rate=config.sampling_rate,
            threshold=config.threshold,
            min_speech_duration_ms=config.min_speech_duration_ms,
            min_silence_duration_ms=config.min_silence_duration_ms,
            window_size_samples=config.window_size_samples
        )
        
        # Convert timestamps to segments
        segments = []
        overall_confidence = 0.0
        
        for segment in speech_timestamps:
            start_sec = segment['start'] / config.sampling_rate
            end_sec = segment['end'] / config.sampling_rate
            confidence = segment.get('confidence', config.threshold)
            
            segments.append(VADSegment(
                start=start_sec,
                end=end_sec,
                confidence=confidence
            ))
            
            overall_confidence = max(overall_confidence, confidence)
        
        # Determine if voice was detected
        has_voice = len(segments) > 0
        
        # If no segments but we still want to provide some confidence metric
        if not has_voice:
            # Calculate simple energy-based confidence as fallback
            if len(wav) > 0:
                energy = torch.mean(wav ** 2).item()
                overall_confidence = min(energy * 10, 1.0)  # Scale energy to 0-1 range
        
        processing_time_ms = (time.time() - start_time) * 1000
        
        logger.info(f"VAD processed {audio_duration_ms:.1f}ms audio in {processing_time_ms:.1f}ms, "
                   f"found {len(segments)} voice segments")
        
        return VADResponse(
            has_voice=has_voice,
            confidence=overall_confidence,
            segments=segments,
            processing_time_ms=processing_time_ms,
            audio_duration_ms=audio_duration_ms
        )
        
    except Exception as e:
        logger.error(f"Error processing audio: {e}")
        raise HTTPException(status_code=500, detail=f"Audio processing failed: {str(e)}")

# Initialize FastAPI app
app = FastAPI(
    title="System Models Service",
    description="Lightweight ML models service including VAD, speech enhancement, and other system-level models",
    version="1.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.on_event("startup")
async def startup_event():
    """Load the model on startup"""
    success = load_silero_model()
    if not success:
        logger.error("Failed to load VAD model on startup")
        # Note: In production, you might want to exit here
        # For development, we'll continue without the model

@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint"""
    return HealthResponse(
        status="healthy" if model is not None else "model_not_loaded",
        model_loaded=model is not None,
        version="1.0.0"
    )

@app.post("/vad", response_model=VADResponse)
async def vad_endpoint(
    file: bytes = File(..., description="Audio file in bytes (WAV, MP3, etc.)"),
    threshold: float = 0.5,
    min_speech_duration_ms: int = 250,
    min_silence_duration_ms: int = 100,
    window_size_samples: int = 1536,
    sampling_rate: int = 16000
):
    """
    Voice Activity Detection endpoint
    
    Accepts audio file bytes and returns voice activity segments.
    """
    config = VADConfig(
        threshold=threshold,
        min_speech_duration_ms=min_speech_duration_ms,
        min_silence_duration_ms=min_silence_duration_ms,
        window_size_samples=window_size_samples,
        sampling_rate=sampling_rate
    )
    
    return process_audio_bytes(file, config)

@app.post("/vad/stream", response_model=VADResponse)
async def vad_stream_endpoint(
    file: UploadFile = File(..., description="Audio file upload"),
    threshold: float = 0.5,
    min_speech_duration_ms: int = 250,
    min_silence_duration_ms: int = 100,
    window_size_samples: int = 1536,
    sampling_rate: int = 16000
):
    """
    Voice Activity Detection endpoint for file uploads
    
    Alternative endpoint that accepts file uploads instead of raw bytes.
    Useful for testing with curl or browser uploads.
    """
    # Read file contents
    audio_bytes = await file.read()
    
    config = VADConfig(
        threshold=threshold,
        min_speech_duration_ms=min_speech_duration_ms,
        min_silence_duration_ms=min_silence_duration_ms,
        window_size_samples=window_size_samples,
        sampling_rate=sampling_rate
    )
    
    return process_audio_bytes(audio_bytes, config)

@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "System Models Service",
        "version": "1.0.0",
        "models": {
            "vad": "Silero VAD - Voice Activity Detection"
        },
        "endpoints": {
            "health": "/health",
            "vad": "/vad",
            "vad_stream": "/vad/stream",
            "docs": "/docs"
        }
    }

if __name__ == "__main__":
    # Run the server
    uvicorn.run(
        "models_server:app",
        host="0.0.0.0",
        port=8001,
        reload=True,
        log_level="info"
    )
