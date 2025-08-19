"""
Audio Interface - Speech-to-text, text-to-speech, and audio processing.
Handles all audio communications with hardware and users.
"""

from typing import Dict, Any, List, Optional, Union
from datetime import datetime
from dataclasses import dataclass
from enum import Enum
import asyncio
import json
import uuid
from pathlib import Path
import io
import wave
import threading
from concurrent.futures import ThreadPoolExecutor

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


class AudioFormat(Enum):
    """Supported audio formats."""
    WAV = "wav"
    MP3 = "mp3"
    FLAC = "flac"
    OGG = "ogg"


class SpeechEngine(Enum):
    """Available speech engines."""
    WHISPER = "whisper"
    GOOGLE = "google"
    AZURE = "azure"
    AMAZON = "amazon"


class TTSEngine(Enum):
    """Available TTS engines."""
    PYTTSX3 = "pyttsx3"
    GOOGLE = "google"
    AZURE = "azure"
    AMAZON = "amazon"
    ELEVENLABS = "elevenlabs"


@dataclass
class AudioProcessingResult:
    """Result from audio processing."""
    success: bool
    text: Optional[str]
    confidence: float
    processing_time: float
    language: str
    error: Optional[str]
    metadata: Dict[str, Any]


@dataclass
class TTSResult:
    """Result from text-to-speech processing."""
    success: bool
    audio_data: Optional[bytes]
    audio_format: AudioFormat
    duration: float
    processing_time: float
    error: Optional[str]
    metadata: Dict[str, Any]


class AudioInterface:
    """
    Audio interface for speech processing and hardware communication.
    Handles STT, TTS, and audio I/O operations.
    """
    
    def __init__(self):
        self.stt_engine = None
        self.tts_engine = None
        self.is_initialized = False
        self.is_recording = False
        self.is_playing = False
        self.executor = ThreadPoolExecutor(max_workers=4)
        self.audio_cache: Dict[str, bytes] = {}
        self.processing_queue = asyncio.Queue()
        self.output_queue: Dict[str, TTSResult] = {}
        
        # Audio settings
        self.sample_rate = settings.audio.sample_rate
        self.channels = settings.audio.channels
        self.chunk_size = 1024
        self.audio_format = "wav"
        
        # Data directories
        self.audio_data_path = Path("data/audio")
        
    async def initialize(self) -> None:
        """Initialize audio interface and engines."""
        logger.info("Initializing Audio Interface")
        
        try:
            # Create data directories
            self.audio_data_path.mkdir(parents=True, exist_ok=True)
            (self.audio_data_path / "input").mkdir(exist_ok=True)
            (self.audio_data_path / "output").mkdir(exist_ok=True)
            (self.audio_data_path / "cache").mkdir(exist_ok=True)
            
            # Initialize STT engine
            await self._initialize_stt_engine()
            
            # Initialize TTS engine
            await self._initialize_tts_engine()
            
            # Start processing worker
            asyncio.create_task(self._processing_worker())
            
            self.is_initialized = True
            logger.info("Audio Interface initialized successfully")
            
        except Exception as e:
            logger.error("Failed to initialize Audio Interface", error=str(e))
            raise
    
    async def process_input(
        self,
        audio_data: Dict[str, Any],
        correlation_id: Optional[str] = None
    ) -> AudioProcessingResult:
        """
        Process audio input from hardware.
        
        Args:
            audio_data: Audio data dictionary
            correlation_id: Optional correlation ID
        
        Returns:
            Audio processing result
        """
        start_time = datetime.now()
        correlation_id = correlation_id or str(uuid.uuid4())
        
        with with_correlation_id(correlation_id):
            logger.info("Processing audio input", 
                       data_type=type(audio_data.get("data")),
                       format=audio_data.get("format"))
            
            try:
                # Extract audio data
                raw_audio = audio_data.get("data")
                audio_format = audio_data.get("format", "wav")
                language = audio_data.get("language", "en")
                
                if not raw_audio:
                    return AudioProcessingResult(
                        success=False,
                        text=None,
                        confidence=0.0,
                        processing_time=0.0,
                        language=language,
                        error="No audio data provided",
                        metadata={"correlation_id": correlation_id}
                    )
                
                # Convert to appropriate format if needed
                processed_audio = await self._preprocess_audio(raw_audio, audio_format)
                
                # Save input audio for debugging
                await self._save_input_audio(processed_audio, correlation_id)
                
                # Perform speech-to-text
                transcription_result = await self._speech_to_text(
                    processed_audio, language
                )
                
                processing_time = (datetime.now() - start_time).total_seconds()
                
                result = AudioProcessingResult(
                    success=transcription_result["success"],
                    text=transcription_result.get("text"),
                    confidence=transcription_result.get("confidence", 0.0),
                    processing_time=processing_time,
                    language=transcription_result.get("language", language),
                    error=transcription_result.get("error"),
                    metadata={
                        "correlation_id": correlation_id,
                        "original_format": audio_format,
                        "engine_used": self._get_stt_engine_name()
                    }
                )
                
                logger.info("Audio processing completed",
                           success=result.success,
                           text_length=len(result.text) if result.text else 0,
                           confidence=result.confidence,
                           processing_time=processing_time)
                
                return result
                
            except Exception as e:
                processing_time = (datetime.now() - start_time).total_seconds()
                logger.error("Audio processing failed", error=str(e))
                
                return AudioProcessingResult(
                    success=False,
                    text=None,
                    confidence=0.0,
                    processing_time=processing_time,
                    language="en",
                    error=str(e),
                    metadata={"correlation_id": correlation_id}
                )
    
    async def generate_speech(
        self,
        text: str,
        voice_settings: Optional[Dict[str, Any]] = None,
        correlation_id: Optional[str] = None
    ) -> TTSResult:
        """
        Generate speech audio from text.
        
        Args:
            text: Text to convert to speech
            voice_settings: Voice configuration
            correlation_id: Optional correlation ID
        
        Returns:
            TTS result
        """
        start_time = datetime.now()
        correlation_id = correlation_id or str(uuid.uuid4())
        
        with with_correlation_id(correlation_id):
            logger.info("Generating speech", 
                       text_length=len(text),
                       text_preview=text[:50] + "..." if len(text) > 50 else text)
            
            try:
                if not text or not text.strip():
                    return TTSResult(
                        success=False,
                        audio_data=None,
                        audio_format=AudioFormat.WAV,
                        duration=0.0,
                        processing_time=0.0,
                        error="No text provided",
                        metadata={"correlation_id": correlation_id}
                    )
                
                # Check cache first
                cache_key = self._get_tts_cache_key(text, voice_settings)
                if cache_key in self.audio_cache:
                    logger.debug("Using cached TTS result")
                    audio_data = self.audio_cache[cache_key]
                    duration = await self._get_audio_duration(audio_data)
                    
                    return TTSResult(
                        success=True,
                        audio_data=audio_data,
                        audio_format=AudioFormat.WAV,
                        duration=duration,
                        processing_time=(datetime.now() - start_time).total_seconds(),
                        error=None,
                        metadata={
                            "correlation_id": correlation_id,
                            "cached": True,
                            "engine_used": self._get_tts_engine_name()
                        }
                    )
                
                # Generate speech
                tts_result = await self._text_to_speech(text, voice_settings)
                
                processing_time = (datetime.now() - start_time).total_seconds()
                
                result = TTSResult(
                    success=tts_result["success"],
                    audio_data=tts_result.get("audio_data"),
                    audio_format=AudioFormat(tts_result.get("format", "wav")),
                    duration=tts_result.get("duration", 0.0),
                    processing_time=processing_time,
                    error=tts_result.get("error"),
                    metadata={
                        "correlation_id": correlation_id,
                        "cached": False,
                        "engine_used": self._get_tts_engine_name(),
                        "voice_settings": voice_settings
                    }
                )
                
                # Cache successful results
                if result.success and result.audio_data:
                    self.audio_cache[cache_key] = result.audio_data
                    
                    # Save to disk cache
                    await self._save_tts_cache(cache_key, result.audio_data)
                
                logger.info("Speech generation completed",
                           success=result.success,
                           duration=result.duration,
                           processing_time=processing_time)
                
                return result
                
            except Exception as e:
                processing_time = (datetime.now() - start_time).total_seconds()
                logger.error("Speech generation failed", error=str(e))
                
                return TTSResult(
                    success=False,
                    audio_data=None,
                    audio_format=AudioFormat.WAV,
                    duration=0.0,
                    processing_time=processing_time,
                    error=str(e),
                    metadata={"correlation_id": correlation_id}
                )
    
    async def get_output(self, response_id: str) -> Dict[str, Any]:
        """
        Get audio output for hardware playback.
        
        Args:
            response_id: Response ID
        
        Returns:
            Audio output data
        """
        try:
            if response_id in self.output_queue:
                tts_result = self.output_queue[response_id]
                
                # Remove from queue after retrieval
                del self.output_queue[response_id]
                
                if tts_result.success and tts_result.audio_data:
                    return {
                        "success": True,
                        "audio_data": tts_result.audio_data,
                        "format": tts_result.audio_format.value,
                        "duration": tts_result.duration,
                        "metadata": tts_result.metadata
                    }
                else:
                    return {
                        "success": False,
                        "error": tts_result.error or "TTS generation failed"
                    }
            
            else:
                return {
                    "success": False,
                    "error": f"Response ID {response_id} not found"
                }
        
        except Exception as e:
            logger.error("Failed to get audio output", error=str(e))
            return {
                "success": False,
                "error": str(e)
            }
    
    async def queue_response_audio(
        self,
        response_id: str,
        text: str,
        voice_settings: Optional[Dict[str, Any]] = None
    ) -> bool:
        """
        Queue audio response for hardware retrieval.
        
        Args:
            response_id: Response ID
            text: Text to convert to speech
            voice_settings: Voice settings
        
        Returns:
            True if successful
        """
        try:
            tts_result = await self.generate_speech(text, voice_settings)
            self.output_queue[response_id] = tts_result
            
            logger.info("Audio response queued", 
                       response_id=response_id,
                       success=tts_result.success)
            
            return tts_result.success
            
        except Exception as e:
            logger.error("Failed to queue audio response", error=str(e))
            return False
    
    async def cleanup(self) -> None:
        """Cleanup audio interface resources."""
        logger.info("Cleaning up Audio Interface")
        
        try:
            self.is_recording = False
            self.is_playing = False
            
            # Stop any ongoing operations
            if hasattr(self, 'recording_thread'):
                self.recording_thread.join(timeout=2.0)
            
            # Cleanup engines
            if self.tts_engine:
                await self._cleanup_tts_engine()
            
            if self.stt_engine:
                await self._cleanup_stt_engine()
            
            # Shutdown executor
            self.executor.shutdown(wait=True)
            
            logger.info("Audio Interface cleanup completed")
            
        except Exception as e:
            logger.error("Audio Interface cleanup failed", error=str(e))
    
    async def _initialize_stt_engine(self) -> None:
        """Initialize speech-to-text engine."""
        try:
            engine_type = getattr(settings.audio, 'stt_engine', 'whisper')
            
            if engine_type == "whisper":
                # Initialize Whisper
                try:
                    import whisper
                    self.stt_engine = whisper.load_model(settings.audio.whisper_model)
                    logger.info("Whisper STT engine initialized", model=settings.audio.whisper_model)
                except ImportError:
                    logger.warning("Whisper not available, falling back to speech_recognition")
                    await self._initialize_speech_recognition()
            else:
                await self._initialize_speech_recognition()
            
        except Exception as e:
            logger.error("Failed to initialize STT engine", error=str(e))
            # Fallback to speech_recognition
            await self._initialize_speech_recognition()
    
    async def _initialize_speech_recognition(self) -> None:
        """Initialize speech_recognition library."""
        try:
            import speech_recognition as sr
            self.stt_engine = sr.Recognizer()
            logger.info("SpeechRecognition STT engine initialized")
        except ImportError:
            logger.error("speech_recognition library not available")
            self.stt_engine = None
    
    async def _initialize_tts_engine(self) -> None:
        """Initialize text-to-speech engine."""
        try:
            engine_type = getattr(settings.audio, 'tts_engine', 'pyttsx3')
            
            if engine_type == "pyttsx3":
                # Initialize pyttsx3
                try:
                    import pyttsx3
                    loop = asyncio.get_event_loop()
                    self.tts_engine = await loop.run_in_executor(
                        self.executor,
                        pyttsx3.init
                    )
                    logger.info("pyttsx3 TTS engine initialized")
                except ImportError:
                    logger.warning("pyttsx3 not available")
                    self.tts_engine = None
            else:
                logger.warning(f"TTS engine {engine_type} not implemented")
                self.tts_engine = None
                
        except Exception as e:
            logger.error("Failed to initialize TTS engine", error=str(e))
            self.tts_engine = None
    
    async def _preprocess_audio(self, audio_data: bytes, format: str) -> bytes:
        """Preprocess audio data."""
        # For now, return as-is
        # In a real implementation, you'd handle format conversion, noise reduction, etc.
        return audio_data
    
    async def _speech_to_text(self, audio_data: bytes, language: str) -> Dict[str, Any]:
        """Convert speech to text."""
        try:
            if not self.stt_engine:
                return {
                    "success": False,
                    "error": "STT engine not initialized"
                }
            
            # Check if we're using Whisper
            if hasattr(self.stt_engine, 'transcribe'):
                # Whisper
                loop = asyncio.get_event_loop()
                result = await loop.run_in_executor(
                    self.executor,
                    self._transcribe_with_whisper,
                    audio_data,
                    language
                )
                return result
            
            else:
                # speech_recognition
                result = await self._transcribe_with_speech_recognition(
                    audio_data, language
                )
                return result
                
        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }
    
    def _transcribe_with_whisper(self, audio_data: bytes, language: str) -> Dict[str, Any]:
        """Transcribe using Whisper (runs in executor)."""
        try:
            # Save audio to temporary file
            import tempfile
            with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp_file:
                tmp_file.write(audio_data)
                tmp_path = tmp_file.name
            
            # Transcribe
            result = self.stt_engine.transcribe(tmp_path, language=language)
            
            # Clean up
            Path(tmp_path).unlink()
            
            return {
                "success": True,
                "text": result["text"].strip(),
                "confidence": 0.9,  # Whisper doesn't provide confidence scores
                "language": result.get("language", language)
            }
            
        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }
    
    async def _transcribe_with_speech_recognition(
        self,
        audio_data: bytes,
        language: str
    ) -> Dict[str, Any]:
        """Transcribe using speech_recognition library."""
        try:
            import speech_recognition as sr
            
            # Convert audio data to AudioData
            audio_file = io.BytesIO(audio_data)
            with sr.AudioFile(audio_file) as source:
                audio = self.stt_engine.record(source)
            
            # Recognize speech
            try:
                text = self.stt_engine.recognize_google(audio, language=language)
                return {
                    "success": True,
                    "text": text,
                    "confidence": 0.8,  # Approximate confidence
                    "language": language
                }
            except sr.UnknownValueError:
                return {
                    "success": False,
                    "error": "Could not understand audio"
                }
            except sr.RequestError as e:
                return {
                    "success": False,
                    "error": f"Speech recognition service error: {str(e)}"
                }
                
        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }
    
    async def _text_to_speech(
        self,
        text: str,
        voice_settings: Optional[Dict[str, Any]]
    ) -> Dict[str, Any]:
        """Convert text to speech."""
        try:
            if not self.tts_engine:
                return {
                    "success": False,
                    "error": "TTS engine not initialized"
                }
            
            # Use pyttsx3
            loop = asyncio.get_event_loop()
            result = await loop.run_in_executor(
                self.executor,
                self._synthesize_with_pyttsx3,
                text,
                voice_settings
            )
            
            return result
            
        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }
    
    def _synthesize_with_pyttsx3(
        self,
        text: str,
        voice_settings: Optional[Dict[str, Any]]
    ) -> Dict[str, Any]:
        """Synthesize speech using pyttsx3 (runs in executor)."""
        try:
            import tempfile
            
            # Configure voice settings
            if voice_settings:
                if "speed" in voice_settings:
                    self.tts_engine.setProperty('rate', int(voice_settings["speed"] * 200))
                if "volume" in voice_settings:
                    self.tts_engine.setProperty('volume', voice_settings["volume"])
            
            # Save to temporary file
            with tempfile.NamedTemporaryFile(suffix=".wav", delete=False) as tmp_file:
                tmp_path = tmp_file.name
            
            self.tts_engine.save_to_file(text, tmp_path)
            self.tts_engine.runAndWait()
            
            # Read audio data
            with open(tmp_path, 'rb') as f:
                audio_data = f.read()
            
            # Get duration
            duration = self._get_wav_duration(audio_data)
            
            # Clean up
            Path(tmp_path).unlink()
            
            return {
                "success": True,
                "audio_data": audio_data,
                "format": "wav",
                "duration": duration
            }
            
        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }
    
    def _get_wav_duration(self, audio_data: bytes) -> float:
        """Get duration of WAV audio data."""
        try:
            audio_file = io.BytesIO(audio_data)
            with wave.open(audio_file, 'rb') as wav_file:
                frames = wav_file.getnframes()
                sample_rate = wav_file.getframerate()
                duration = frames / float(sample_rate)
                return duration
        except Exception:
            return 0.0
    
    async def _get_audio_duration(self, audio_data: bytes) -> float:
        """Get duration of audio data."""
        # This is a simplified implementation
        # In practice, you'd detect format and use appropriate decoder
        return self._get_wav_duration(audio_data)
    
    async def _save_input_audio(self, audio_data: bytes, correlation_id: str) -> None:
        """Save input audio for debugging."""
        try:
            if settings.debug:
                file_path = self.audio_data_path / "input" / f"{correlation_id}.wav"
                with open(file_path, 'wb') as f:
                    f.write(audio_data)
        except Exception as e:
            logger.error("Failed to save input audio", error=str(e))
    
    async def _save_tts_cache(self, cache_key: str, audio_data: bytes) -> None:
        """Save TTS result to disk cache."""
        try:
            cache_file = self.audio_data_path / "cache" / f"{cache_key}.wav"
            with open(cache_file, 'wb') as f:
                f.write(audio_data)
        except Exception as e:
            logger.error("Failed to save TTS cache", error=str(e))
    
    def _get_tts_cache_key(
        self,
        text: str,
        voice_settings: Optional[Dict[str, Any]]
    ) -> str:
        """Generate cache key for TTS."""
        import hashlib
        
        settings_str = json.dumps(voice_settings or {}, sort_keys=True)
        combined = f"{text}|{settings_str}"
        
        return hashlib.md5(combined.encode()).hexdigest()
    
    def _get_stt_engine_name(self) -> str:
        """Get STT engine name."""
        if hasattr(self.stt_engine, 'transcribe'):
            return "whisper"
        else:
            return "speech_recognition"
    
    def _get_tts_engine_name(self) -> str:
        """Get TTS engine name."""
        return "pyttsx3"
    
    async def _processing_worker(self) -> None:
        """Background worker for processing audio queue."""
        while True:
            try:
                # Process queued audio tasks
                await asyncio.sleep(0.1)
                
                # Clean up old cache entries
                await self._cleanup_cache()
                
            except Exception as e:
                logger.error("Audio processing worker error", error=str(e))
                await asyncio.sleep(1)
    
    async def _cleanup_cache(self) -> None:
        """Clean up old cache entries."""
        try:
            # Keep cache size reasonable
            if len(self.audio_cache) > 100:
                # Remove oldest entries (simplified)
                keys_to_remove = list(self.audio_cache.keys())[:20]
                for key in keys_to_remove:
                    del self.audio_cache[key]
        except Exception as e:
            logger.error("Cache cleanup failed", error=str(e))
    
    async def _cleanup_stt_engine(self) -> None:
        """Cleanup STT engine."""
        try:
            # STT engines typically don't need explicit cleanup
            self.stt_engine = None
        except Exception as e:
            logger.error("STT engine cleanup failed", error=str(e))
    
    async def _cleanup_tts_engine(self) -> None:
        """Cleanup TTS engine."""
        try:
            if hasattr(self.tts_engine, 'stop'):
                self.tts_engine.stop()
            self.tts_engine = None
        except Exception as e:
            logger.error("TTS engine cleanup failed", error=str(e))
    
    def get_status(self) -> Dict[str, Any]:
        """Get audio interface status."""
        return {
            "initialized": self.is_initialized,
            "recording": self.is_recording,
            "playing": self.is_playing,
            "stt_engine": self._get_stt_engine_name() if self.stt_engine else None,
            "tts_engine": self._get_tts_engine_name() if self.tts_engine else None,
            "cache_size": len(self.audio_cache),
            "output_queue_size": len(self.output_queue),
            "sample_rate": self.sample_rate,
            "channels": self.channels
        }
