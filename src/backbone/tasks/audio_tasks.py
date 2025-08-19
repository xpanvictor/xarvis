"""
Audio Tasks - Celery tasks for audio processing and speech operations.
Handles speech-to-text, text-to-speech, and audio file processing.
"""

from typing import Dict, Any, List, Optional, Union
from datetime import datetime
import json
import uuid
import base64
from pathlib import Path

from celery import current_app
from celery.exceptions import Retry

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


@current_app.task(
    bind=True,
    name="audio.process_speech_to_text",
    max_retries=3,
    default_retry_delay=30
)
def process_speech_to_text(
    self,
    audio_data: Union[str, bytes],
    audio_format: str = "wav",
    language: str = "en",
    model: str = "whisper"
) -> Dict[str, Any]:
    """
    Convert speech audio to text.
    
    Args:
        audio_data: Audio data (base64 string or bytes)
        audio_format: Audio format (wav, mp3, etc.)
        language: Audio language
        model: STT model to use
    
    Returns:
        Speech-to-text result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Processing speech to text", 
                   audio_format=audio_format,
                   language=language,
                   model=model)
        
        try:
            # Import here to avoid circular imports
            from src.interfaces.audio_interface import AudioInterface
            
            audio_interface = AudioInterface()
            
            # Convert base64 to bytes if needed
            if isinstance(audio_data, str):
                audio_bytes = base64.b64decode(audio_data)
            else:
                audio_bytes = audio_data
            
            # Process speech to text
            result = audio_interface.speech_to_text(
                audio_data=audio_bytes,
                language=language,
                model=model
            )
            
            logger.info("Speech to text completed", 
                       text_length=len(result.get("text", "")),
                       confidence=result.get("confidence", 0),
                       model=model)
            
            return {
                "success": True,
                "result": result,
                "model": model,
                "language": language,
                "audio_format": audio_format,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Speech to text failed", 
                        model=model,
                        language=language,
                        error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=30, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "model": model,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="audio.generate_text_to_speech",
    max_retries=3,
    default_retry_delay=20
)
def generate_text_to_speech(
    self,
    text: str,
    voice: str = "default",
    speed: float = 1.0,
    language: str = "en",
    output_format: str = "wav"
) -> Dict[str, Any]:
    """
    Convert text to speech audio.
    
    Args:
        text: Text to convert
        voice: Voice to use
        speed: Speech speed multiplier
        language: Output language
        output_format: Audio output format
    
    Returns:
        Text-to-speech result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Generating text to speech", 
                   text_length=len(text),
                   voice=voice,
                   speed=speed,
                   language=language)
        
        try:
            # Import here to avoid circular imports
            from src.interfaces.audio_interface import AudioInterface
            
            audio_interface = AudioInterface()
            
            # Generate speech
            result = audio_interface.text_to_speech(
                text=text,
                voice=voice,
                speed=speed,
                language=language
            )
            
            # Convert audio to base64 for serialization
            if result.get("success") and result.get("audio_data"):
                audio_b64 = base64.b64encode(result["audio_data"]).decode('utf-8')
                result["audio_data"] = audio_b64
                result["audio_size"] = len(result["audio_data"])
            
            logger.info("Text to speech completed", 
                       text_length=len(text),
                       audio_size=result.get("audio_size", 0),
                       voice=voice)
            
            return {
                "success": True,
                "result": result,
                "voice": voice,
                "speed": speed,
                "language": language,
                "output_format": output_format,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Text to speech failed", 
                        voice=voice,
                        text_length=len(text),
                        error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=20, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "voice": voice,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="audio.process_audio_file",
    max_retries=2,
    default_retry_delay=30
)
def process_audio_file(
    self,
    file_path: str,
    processing_type: str = "transcribe",
    options: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Process an audio file with various operations.
    
    Args:
        file_path: Path to audio file
        processing_type: Type of processing (transcribe, enhance, convert)
        options: Processing options
    
    Returns:
        Processing result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Processing audio file", 
                   file_path=file_path,
                   processing_type=processing_type)
        
        try:
            # Import here to avoid circular imports
            from src.interfaces.audio_interface import AudioInterface
            
            audio_interface = AudioInterface()
            file_path_obj = Path(file_path)
            
            if not file_path_obj.exists():
                return {
                    "success": False,
                    "error": f"Audio file not found: {file_path}",
                    "correlation_id": correlation_id,
                    "timestamp": datetime.now().isoformat()
                }
            
            # Process based on type
            if processing_type == "transcribe":
                result = audio_interface.transcribe_file(
                    file_path=file_path,
                    **options or {}
                )
            
            elif processing_type == "enhance":
                result = audio_interface.enhance_audio(
                    file_path=file_path,
                    **options or {}
                )
            
            elif processing_type == "convert":
                result = audio_interface.convert_audio(
                    file_path=file_path,
                    **options or {}
                )
            
            else:
                return {
                    "success": False,
                    "error": f"Unknown processing type: {processing_type}",
                    "correlation_id": correlation_id,
                    "timestamp": datetime.now().isoformat()
                }
            
            logger.info("Audio file processing completed", 
                       file_path=file_path,
                       processing_type=processing_type,
                       success=result.get("success", False))
            
            return {
                "success": True,
                "result": result,
                "file_path": file_path,
                "processing_type": processing_type,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Audio file processing failed", 
                        file_path=file_path,
                        processing_type=processing_type,
                        error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=30, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="audio.batch_audio_processing",
    max_retries=2
)
def batch_audio_processing(
    self,
    audio_files: List[Dict[str, Any]],
    processing_options: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Process multiple audio files in batch.
    
    Args:
        audio_files: List of audio file specifications
        processing_options: Batch processing options
    
    Returns:
        Batch processing results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Processing audio batch", count=len(audio_files))
        
        try:
            results = []
            errors = []
            
            for i, audio_spec in enumerate(audio_files):
                try:
                    # Process individual audio file
                    result = process_audio_file.apply_async(
                        args=[
                            audio_spec.get("file_path", ""),
                            audio_spec.get("processing_type", "transcribe"),
                            audio_spec.get("options")
                        ]
                    ).get(timeout=120)
                    
                    results.append({
                        "index": i,
                        "file_path": audio_spec.get("file_path"),
                        "success": True,
                        "result": result
                    })
                    
                except Exception as e:
                    errors.append({
                        "index": i,
                        "file_path": audio_spec.get("file_path"),
                        "error": str(e)
                    })
            
            logger.info("Batch audio processing completed", 
                       processed=len(results),
                       errors=len(errors))
            
            return {
                "success": True,
                "processed": len(results),
                "errors": len(errors),
                "results": results,
                "error_details": errors,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Batch audio processing failed", error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=60, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="audio.voice_activity_detection",
    max_retries=2
)
def voice_activity_detection(
    self,
    audio_data: Union[str, bytes],
    audio_format: str = "wav",
    sensitivity: float = 0.5
) -> Dict[str, Any]:
    """
    Detect voice activity in audio.
    
    Args:
        audio_data: Audio data (base64 string or bytes)
        audio_format: Audio format
        sensitivity: Detection sensitivity (0.0 - 1.0)
    
    Returns:
        Voice activity detection result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Detecting voice activity", 
                   audio_format=audio_format,
                   sensitivity=sensitivity)
        
        try:
            # Import here to avoid circular imports
            from src.interfaces.audio_interface import AudioInterface
            
            audio_interface = AudioInterface()
            
            # Convert base64 to bytes if needed
            if isinstance(audio_data, str):
                audio_bytes = base64.b64decode(audio_data)
            else:
                audio_bytes = audio_data
            
            # Detect voice activity
            result = audio_interface.detect_voice_activity(
                audio_data=audio_bytes,
                sensitivity=sensitivity
            )
            
            logger.info("Voice activity detection completed", 
                       has_voice=result.get("has_voice", False),
                       confidence=result.get("confidence", 0))
            
            return {
                "success": True,
                "result": result,
                "audio_format": audio_format,
                "sensitivity": sensitivity,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Voice activity detection failed", error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=20, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="audio.audio_enhancement",
    max_retries=2
)
def audio_enhancement(
    self,
    audio_data: Union[str, bytes],
    enhancement_type: str = "noise_reduction",
    options: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Enhance audio quality.
    
    Args:
        audio_data: Audio data (base64 string or bytes)
        enhancement_type: Type of enhancement
        options: Enhancement options
    
    Returns:
        Audio enhancement result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Enhancing audio", 
                   enhancement_type=enhancement_type)
        
        try:
            # Import here to avoid circular imports
            from src.interfaces.audio_interface import AudioInterface
            
            audio_interface = AudioInterface()
            
            # Convert base64 to bytes if needed
            if isinstance(audio_data, str):
                audio_bytes = base64.b64decode(audio_data)
            else:
                audio_bytes = audio_data
            
            # Enhance audio
            result = audio_interface.enhance_audio_data(
                audio_data=audio_bytes,
                enhancement_type=enhancement_type,
                options=options or {}
            )
            
            # Convert result audio to base64
            if result.get("success") and result.get("enhanced_audio"):
                enhanced_b64 = base64.b64encode(result["enhanced_audio"]).decode('utf-8')
                result["enhanced_audio"] = enhanced_b64
                result["audio_size"] = len(result["enhanced_audio"])
            
            logger.info("Audio enhancement completed", 
                       enhancement_type=enhancement_type,
                       success=result.get("success", False))
            
            return {
                "success": True,
                "result": result,
                "enhancement_type": enhancement_type,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Audio enhancement failed", 
                        enhancement_type=enhancement_type,
                        error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=30, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="audio.cleanup_audio_cache",
    max_retries=1
)
def cleanup_audio_cache(
    self,
    older_than_hours: int = 24,
    cache_types: List[str] = None
) -> Dict[str, Any]:
    """
    Clean up audio cache files.
    
    Args:
        older_than_hours: Remove files older than this many hours
        cache_types: Types of cache to clean (stt, tts, processed)
    
    Returns:
        Cleanup result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Cleaning audio cache", 
                   older_than_hours=older_than_hours,
                   cache_types=cache_types)
        
        try:
            # Import here to avoid circular imports
            from src.interfaces.audio_interface import AudioInterface
            
            audio_interface = AudioInterface()
            
            # Clean cache
            result = audio_interface.cleanup_cache(
                older_than_hours=older_than_hours,
                cache_types=cache_types or ["stt", "tts", "processed"]
            )
            
            logger.info("Audio cache cleanup completed", 
                       files_removed=result.get("files_removed", 0),
                       space_freed_mb=result.get("space_freed_mb", 0))
            
            return {
                "success": True,
                "result": result,
                "older_than_hours": older_than_hours,
                "cache_types": cache_types,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Audio cache cleanup failed", error=str(e))
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


# Export tasks
__all__ = [
    "process_speech_to_text",
    "generate_text_to_speech",
    "process_audio_file",
    "batch_audio_processing",
    "voice_activity_detection",
    "audio_enhancement",
    "cleanup_audio_cache"
]
