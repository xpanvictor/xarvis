"""
Conversation API - Real-time conversation endpoints with streaming support.
Supports unicellular conversation model with WebSocket for real-time interaction.
"""

from fastapi import APIRouter, WebSocket, WebSocketDisconnect, HTTPException, Depends
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field
from typing import Dict, Any, List, Optional, AsyncGenerator
from datetime import datetime
import asyncio
import json
import uuid

from src.utils.logging import get_logger, with_correlation_id
from src.backbone.pipeline_server import PipelineServer
from src.config import settings

logger = get_logger(__name__)

# Create router
router = APIRouter(prefix="/conversation", tags=["conversation"])

# Global pipeline server instance
pipeline_server = None

# Connection manager for WebSocket connections
class ConnectionManager:
    def __init__(self):
        self.active_connections: Dict[str, WebSocket] = {}
        self.user_sessions: Dict[str, str] = {}  # user_id -> session_id
    
    async def connect(self, websocket: WebSocket, user_id: str):
        await websocket.accept()
        session_id = self.user_sessions.get(user_id, str(uuid.uuid4()))
        self.active_connections[session_id] = websocket
        self.user_sessions[user_id] = session_id
        logger.info("WebSocket connected", user_id=user_id, session_id=session_id)
        return session_id
    
    def disconnect(self, session_id: str):
        if session_id in self.active_connections:
            del self.active_connections[session_id]
        # Find and remove user mapping
        user_to_remove = None
        for user_id, sess_id in self.user_sessions.items():
            if sess_id == session_id:
                user_to_remove = user_id
                break
        if user_to_remove:
            del self.user_sessions[user_to_remove]
        logger.info("WebSocket disconnected", session_id=session_id)
    
    async def send_message(self, session_id: str, message: dict):
        if session_id in self.active_connections:
            await self.active_connections[session_id].send_json(message)
    
    async def send_streaming_response(self, session_id: str, response_generator):
        """Send streaming response through WebSocket."""
        if session_id in self.active_connections:
            async for chunk in response_generator:
                await self.active_connections[session_id].send_json({
                    "type": "stream_chunk",
                    "data": chunk
                })

manager = ConnectionManager()


class ConversationMessage(BaseModel):
    """Message in conversation."""
    message: str = Field(..., description="User message")
    user_id: str = Field(default="default_user", description="User identifier")
    context: Optional[Dict[str, Any]] = Field(default_factory=dict, description="Additional context")
    audio_data: Optional[str] = Field(None, description="Base64 encoded audio data")
    continue_listening: Optional[bool] = Field(False, description="Whether to continue listening for more audio")


class StreamingConversationMessage(BaseModel):
    """Streaming conversation message."""
    message: str = Field(..., description="User message")
    user_id: str = Field(default="default_user", description="User identifier")
    stream: bool = Field(True, description="Enable streaming response")
    context: Optional[Dict[str, Any]] = Field(default_factory=dict, description="Additional context")


class HardwareAudioRequest(BaseModel):
    """Audio request from hardware."""
    device_id: str = Field(..., description="Hardware device ID")
    audio_data: str = Field(..., description="Base64 encoded audio data")
    duration: float = Field(..., description="Audio duration in seconds")
    continue_listening: bool = Field(False, description="Whether device should continue listening")
    user_id: str = Field(default="default_user", description="User identifier")


@router.post("/message")
async def send_message(message_data: ConversationMessage):
    """
    Send a message in the unicellular conversation.
    This maintains a single ongoing conversation context.
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Processing conversation message", 
                   user_id=message_data.user_id,
                   message_length=len(message_data.message))
        
        try:
            global pipeline_server
            if not pipeline_server:
                pipeline_server = PipelineServer()
                await pipeline_server.initialize()
            
            # Build request context
            request_data = {
                "primary_input": message_data.message,
                "user_id": message_data.user_id,
                "session_id": f"unicellular_{message_data.user_id}",  # Single session per user
                "context": {
                    **message_data.context,
                    "audio_data": message_data.audio_data,
                    "continue_listening": message_data.continue_listening,
                    "timestamp": datetime.now().isoformat()
                },
                "correlation_id": correlation_id
            }
            
            # Process through pipeline
            result = await pipeline_server.process_request(
                request_data=request_data,
                request_type="conversation_message",
                correlation_id=correlation_id
            )
            
            # Handle continue listening for hardware
            if message_data.continue_listening and result.get("success"):
                # Schedule background task to handle extended listening
                asyncio.create_task(handle_extended_listening(
                    message_data.user_id, 
                    correlation_id,
                    result
                ))
            
            return {
                "success": result.get("success", False),
                "response": result.get("response", "I'm having trouble processing your request."),
                "conversation_id": f"unicellular_{message_data.user_id}",
                "correlation_id": correlation_id,
                "audio_response": result.get("audio_response"),
                "actions": result.get("actions", []),
                "continue_listening": message_data.continue_listening,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Conversation message processing failed", error=str(e))
            return {
                "success": False,
                "error": str(e),
                "conversation_id": f"unicellular_{message_data.user_id}",
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@router.post("/stream")
async def stream_conversation(message_data: StreamingConversationMessage):
    """
    Stream conversation response in real-time.
    Returns SSE (Server-Sent Events) for streaming.
    """
    correlation_id = str(uuid.uuid4())
    
    async def generate_response():
        with with_correlation_id(correlation_id):
            try:
                global pipeline_server
                if not pipeline_server:
                    pipeline_server = PipelineServer()
                    await pipeline_server.initialize()
                
                # Build request context
                request_data = {
                    "primary_input": message_data.message,
                    "user_id": message_data.user_id,
                    "session_id": f"unicellular_{message_data.user_id}",
                    "context": {
                        **message_data.context,
                        "streaming": True,
                        "timestamp": datetime.now().isoformat()
                    },
                    "correlation_id": correlation_id
                }
                
                # Send initial status
                yield f"data: {json.dumps({'type': 'start', 'correlation_id': correlation_id})}\n\n"
                
                # Process through pipeline with streaming
                async for chunk in pipeline_server.process_request_stream(
                    request_data=request_data,
                    request_type="conversation_message",
                    correlation_id=correlation_id
                ):
                    yield f"data: {json.dumps(chunk)}\n\n"
                
                # Send completion
                yield f"data: {json.dumps({'type': 'complete', 'correlation_id': correlation_id})}\n\n"
                
            except Exception as e:
                logger.error("Streaming conversation failed", error=str(e))
                yield f"data: {json.dumps({'type': 'error', 'error': str(e)})}\n\n"
    
    return StreamingResponse(
        generate_response(),
        media_type="text/plain",
        headers={"Cache-Control": "no-cache", "Connection": "keep-alive"}
    )


@router.websocket("/ws/{user_id}")
async def websocket_conversation(websocket: WebSocket, user_id: str):
    """
    WebSocket endpoint for real-time conversation.
    Maintains persistent connection for continuous interaction.
    """
    session_id = await manager.connect(websocket, user_id)
    
    try:
        global pipeline_server
        if not pipeline_server:
            pipeline_server = PipelineServer()
            await pipeline_server.initialize()
        
        # Send connection confirmation
        await manager.send_message(session_id, {
            "type": "connected",
            "session_id": session_id,
            "user_id": user_id,
            "message": "Connected to Jarvis. How can I assist you?"
        })
        
        while True:
            # Receive message from client
            data = await websocket.receive_json()
            correlation_id = str(uuid.uuid4())
            
            with with_correlation_id(correlation_id):
                logger.info("WebSocket message received", 
                           user_id=user_id,
                           session_id=session_id,
                           message_type=data.get("type"))
                
                if data.get("type") == "message":
                    # Process conversation message
                    request_data = {
                        "primary_input": data.get("message", ""),
                        "user_id": user_id,
                        "session_id": session_id,
                        "context": {
                            **data.get("context", {}),
                            "websocket": True,
                            "timestamp": datetime.now().isoformat()
                        },
                        "correlation_id": correlation_id
                    }
                    
                    # Send thinking indicator
                    await manager.send_message(session_id, {
                        "type": "thinking",
                        "correlation_id": correlation_id
                    })
                    
                    # Process request
                    if data.get("stream", True):
                        # Stream response
                        await manager.send_message(session_id, {
                            "type": "stream_start",
                            "correlation_id": correlation_id
                        })
                        
                        async for chunk in pipeline_server.process_request_stream(
                            request_data=request_data,
                            request_type="conversation_message",
                            correlation_id=correlation_id
                        ):
                            await manager.send_message(session_id, {
                                "type": "stream_chunk",
                                "data": chunk,
                                "correlation_id": correlation_id
                            })
                        
                        await manager.send_message(session_id, {
                            "type": "stream_complete",
                            "correlation_id": correlation_id
                        })
                    else:
                        # Send complete response
                        result = await pipeline_server.process_request(
                            request_data=request_data,
                            request_type="conversation_message",
                            correlation_id=correlation_id
                        )
                        
                        await manager.send_message(session_id, {
                            "type": "response",
                            "data": result,
                            "correlation_id": correlation_id
                        })
                
                elif data.get("type") == "audio":
                    # Handle audio input
                    await handle_audio_input(session_id, user_id, data, correlation_id)
                
                elif data.get("type") == "ping":
                    # Handle ping/pong for connection health
                    await manager.send_message(session_id, {
                        "type": "pong",
                        "timestamp": datetime.now().isoformat()
                    })
    
    except WebSocketDisconnect:
        manager.disconnect(session_id)
        logger.info("WebSocket disconnected", user_id=user_id, session_id=session_id)
    except Exception as e:
        logger.error("WebSocket error", error=str(e), user_id=user_id)
        manager.disconnect(session_id)


@router.post("/hardware/audio")
async def handle_hardware_audio(audio_request: HardwareAudioRequest):
    """
    Handle audio input from hardware devices (ESP32, etc.).
    Supports continuous conversation flow with hardware.
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Processing hardware audio", 
                   device_id=audio_request.device_id,
                   user_id=audio_request.user_id,
                   duration=audio_request.duration)
        
        try:
            global pipeline_server
            if not pipeline_server:
                pipeline_server = PipelineServer()
                await pipeline_server.initialize()
            
            # Process audio through speech-to-text first
            from src.interfaces.audio_interface import AudioInterface
            audio_interface = AudioInterface()
            
            # Convert base64 audio to text
            import base64
            audio_bytes = base64.b64decode(audio_request.audio_data)
            
            stt_result = await audio_interface.speech_to_text(audio_bytes)
            
            if not stt_result.get("success"):
                return {
                    "success": False,
                    "error": "Failed to transcribe audio",
                    "device_id": audio_request.device_id,
                    "correlation_id": correlation_id
                }
            
            transcribed_text = stt_result.get("text", "")
            
            # Process transcribed text through conversation pipeline
            request_data = {
                "primary_input": transcribed_text,
                "user_id": audio_request.user_id,
                "session_id": f"unicellular_{audio_request.user_id}",
                "context": {
                    "device_id": audio_request.device_id,
                    "audio_duration": audio_request.duration,
                    "continue_listening": audio_request.continue_listening,
                    "transcription_confidence": stt_result.get("confidence", 0.0),
                    "hardware_input": True,
                    "timestamp": datetime.now().isoformat()
                },
                "correlation_id": correlation_id
            }
            
            result = await pipeline_server.process_request(
                request_data=request_data,
                request_type="hardware_conversation",
                correlation_id=correlation_id
            )
            
            # Generate audio response if successful
            response_audio = None
            if result.get("success") and result.get("response"):
                tts_result = await audio_interface.text_to_speech(result["response"])
                if tts_result.get("success"):
                    response_audio = base64.b64encode(tts_result["audio_data"]).decode('utf-8')
            
            # Determine if we should continue listening
            should_continue = (
                audio_request.continue_listening and 
                result.get("success") and
                not result.get("conversation_complete", False)
            )
            
            return {
                "success": result.get("success", False),
                "transcribed_text": transcribed_text,
                "response": result.get("response", ""),
                "response_audio": response_audio,
                "continue_listening": should_continue,
                "listening_duration": 5.0 if should_continue else 0.0,
                "device_id": audio_request.device_id,
                "correlation_id": correlation_id,
                "actions": result.get("actions", []),
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Hardware audio processing failed", error=str(e))
            return {
                "success": False,
                "error": str(e),
                "device_id": audio_request.device_id,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


async def handle_audio_input(session_id: str, user_id: str, audio_data: dict, correlation_id: str):
    """Handle audio input through WebSocket."""
    try:
        # Similar to hardware audio processing but through WebSocket
        from src.interfaces.audio_interface import AudioInterface
        audio_interface = AudioInterface()
        
        # Process audio
        import base64
        audio_bytes = base64.b64decode(audio_data.get("audio_data", ""))
        
        stt_result = await audio_interface.speech_to_text(audio_bytes)
        
        if stt_result.get("success"):
            # Send transcription
            await manager.send_message(session_id, {
                "type": "transcription",
                "text": stt_result.get("text", ""),
                "confidence": stt_result.get("confidence", 0.0),
                "correlation_id": correlation_id
            })
            
            # Process as regular message
            request_data = {
                "primary_input": stt_result.get("text", ""),
                "user_id": user_id,
                "session_id": session_id,
                "context": {
                    "audio_input": True,
                    "transcription_confidence": stt_result.get("confidence", 0.0),
                    "websocket": True,
                    "timestamp": datetime.now().isoformat()
                },
                "correlation_id": correlation_id
            }
            
            global pipeline_server
            result = await pipeline_server.process_request(
                request_data=request_data,
                request_type="conversation_message",
                correlation_id=correlation_id
            )
            
            await manager.send_message(session_id, {
                "type": "audio_response",
                "data": result,
                "correlation_id": correlation_id
            })
    
    except Exception as e:
        logger.error("Audio input processing failed", error=str(e))
        await manager.send_message(session_id, {
            "type": "error",
            "error": str(e),
            "correlation_id": correlation_id
        })


async def handle_extended_listening(user_id: str, correlation_id: str, previous_result: dict):
    """
    Handle extended listening scenarios where processing is ongoing.
    This allows for dynamic conversation flow.
    """
    logger.info("Handling extended listening", user_id=user_id, correlation_id=correlation_id)
    
    # This could trigger hardware to extend listening time
    # or prepare for follow-up responses based on context
    
    # Example: If the AI is thinking about a complex response,
    # we can signal the hardware to continue listening for clarification
    
    # Implementation would depend on specific hardware communication protocol
    pass


@router.get("/status/{user_id}")
async def get_conversation_status(user_id: str):
    """Get status of user's conversation."""
    session_id = f"unicellular_{user_id}"
    
    # Get session information from pipeline server
    global pipeline_server
    if pipeline_server:
        status = await pipeline_server.get_session_status(session_id)
    else:
        status = {"status": "inactive"}
    
    return {
        "user_id": user_id,
        "session_id": session_id,
        "status": status,
        "websocket_connected": user_id in manager.user_sessions,
        "timestamp": datetime.now().isoformat()
    }


@router.post("/reset/{user_id}")
async def reset_conversation(user_id: str):
    """Reset user's conversation context while maintaining continuity."""
    session_id = f"unicellular_{user_id}"
    
    logger.info("Resetting conversation context", user_id=user_id, session_id=session_id)
    
    global pipeline_server
    if pipeline_server:
        await pipeline_server.reset_session(session_id)
    
    return {
        "success": True,
        "user_id": user_id,
        "session_id": session_id,
        "message": "Conversation context reset while maintaining continuity",
        "timestamp": datetime.now().isoformat()
    }
