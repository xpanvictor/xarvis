"""
Pipeline Server - Central coordination hub for all system communications.
Handles incoming requests, coordinates with other components, and manages responses.
"""

import asyncio
from typing import Dict, Any, Optional, List, AsyncGenerator
from datetime import datetime
from dataclasses import dataclass
import uuid
from enum import Enum
import json

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


class RequestType(Enum):
    """Types of requests the pipeline can handle."""
    AUDIO_INPUT = "audio_input"
    TEXT_INPUT = "text_input"
    SYSTEM_COMMAND = "system_command"
    BACKGROUND_TASK = "background_task"


@dataclass
class PipelineRequest:
    """Represents a request through the pipeline."""
    id: str
    type: RequestType
    data: Dict[str, Any]
    metadata: Dict[str, Any]
    timestamp: datetime
    correlation_id: str


@dataclass
class PipelineResponse:
    """Represents a response from the pipeline."""
    request_id: str
    success: bool
    data: Dict[str, Any]
    error: Optional[str]
    processing_time: float
    timestamp: datetime


class PipelineServer:
    """
    Central pipeline server that coordinates all system operations.
    Acts as the main entry point and orchestrates communication between components.
    """
    
    def __init__(self, brain, job_runner, aggregator):
        self.brain = brain
        self.job_runner = job_runner
        self.aggregator = aggregator
        self.is_running = False
        self.active_requests: Dict[str, PipelineRequest] = {}
        self.response_handlers: Dict[str, asyncio.Future] = {}
        
    async def start(self) -> None:
        """Start the pipeline server."""
        logger.info("Starting Pipeline Server")
        self.is_running = True
        
        # Start background processing loop
        asyncio.create_task(self._process_loop())
        
    async def stop(self) -> None:
        """Stop the pipeline server."""
        logger.info("Stopping Pipeline Server")
        self.is_running = False
        
        # Cancel any pending requests
        for future in self.response_handlers.values():
            if not future.done():
                future.cancel()
    
    async def process_input(
        self,
        text: Optional[str] = None,
        audio: Optional[bytes] = None,
        metadata: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """
        Process input through the pipeline.
        
        Args:
            text: Text input from user
            audio: Audio data from hardware
            metadata: Additional request metadata
        
        Returns:
            Processed response
        """
        start_time = datetime.now()
        correlation_id = str(uuid.uuid4())
        
        with with_correlation_id(correlation_id):
            logger.info("Processing input request", text=bool(text), audio=bool(audio))
            
            try:
                # Create request
                request = PipelineRequest(
                    id=str(uuid.uuid4()),
                    type=RequestType.AUDIO_INPUT if audio else RequestType.TEXT_INPUT,
                    data={"text": text, "audio": audio},
                    metadata=metadata or {},
                    timestamp=start_time,
                    correlation_id=correlation_id
                )
                
                # Store active request
                self.active_requests[request.id] = request
                
                # Process request
                response_data = await self._process_request(request)
                
                # Calculate processing time
                processing_time = (datetime.now() - start_time).total_seconds()
                
                # Create response
                response = PipelineResponse(
                    request_id=request.id,
                    success=True,
                    data=response_data,
                    error=None,
                    processing_time=processing_time,
                    timestamp=datetime.now()
                )
                
                logger.info("Request processed successfully", 
                           request_id=request.id, 
                           processing_time=processing_time)
                
                return self._format_response(response)
                
            except Exception as e:
                processing_time = (datetime.now() - start_time).total_seconds()
                logger.error("Request processing failed", 
                           error=str(e), 
                           processing_time=processing_time)
                
                error_response = PipelineResponse(
                    request_id=correlation_id,
                    success=False,
                    data={},
                    error=str(e),
                    processing_time=processing_time,
                    timestamp=datetime.now()
                )
                
                return self._format_response(error_response)
            
            finally:
                # Cleanup
                if request.id in self.active_requests:
                    del self.active_requests[request.id]
    
    async def _process_request(self, request: PipelineRequest) -> Dict[str, Any]:
        """
        Process a pipeline request through the appropriate components.
        
        Args:
            request: The pipeline request to process
        
        Returns:
            Processed response data
        """
        # Step 1: Aggregate and preprocess input
        aggregated_data = await self.aggregator.process(request.data, request.metadata)
        
        # Step 2: Submit to brain for processing
        brain_response = await self.brain.process(
            input_data=aggregated_data,
            request_type=request.type.value,
            correlation_id=request.correlation_id
        )
        
        # Step 3: Handle any background tasks
        if brain_response.get("background_tasks"):
            for task in brain_response["background_tasks"]:
                await self.job_runner.submit_task(task)
        
        # Step 4: Format and return response
        return {
            "response": brain_response.get("response", ""),
            "audio_response": brain_response.get("audio_response"),
            "actions": brain_response.get("actions", []),
            "context": brain_response.get("context", {}),
            "conversation_id": brain_response.get("conversation_id"),
            "metadata": brain_response.get("metadata", {})
        }
    
    async def _process_loop(self) -> None:
        """Background processing loop for system maintenance."""
        while self.is_running:
            try:
                # Cleanup old requests
                await self._cleanup_old_requests()
                
                # Process any queued system tasks
                await self._process_system_tasks()
                
                await asyncio.sleep(1)  # Process every second
                
            except Exception as e:
                logger.error("Error in processing loop", error=str(e))
                await asyncio.sleep(5)  # Wait longer on error
    
    async def _cleanup_old_requests(self) -> None:
        """Remove old requests from active tracking."""
        current_time = datetime.now()
        expired_requests = []
        
        for request_id, request in self.active_requests.items():
            if (current_time - request.timestamp).total_seconds() > 300:  # 5 minutes
                expired_requests.append(request_id)
        
        for request_id in expired_requests:
            del self.active_requests[request_id]
            logger.debug("Cleaned up expired request", request_id=request_id)
    
    async def _process_system_tasks(self) -> None:
        """Process any pending system-level tasks."""
        # This could include:
        # - Memory cleanup
        # - Model updates
        # - Health checks
        # - Performance monitoring
        pass
    
    def _format_response(self, response: PipelineResponse) -> Dict[str, Any]:
        """
        Format pipeline response for API return.
        
        Args:
            response: Pipeline response object
        
        Returns:
            Formatted response dictionary
        """
        return {
            "success": response.success,
            "data": response.data,
            "error": response.error,
            "processing_time": response.processing_time,
            "timestamp": response.timestamp.isoformat(),
            "request_id": response.request_id
        }
    
    def get_status(self) -> Dict[str, Any]:
        """Get current pipeline server status."""
        return {
            "running": self.is_running,
            "active_requests": len(self.active_requests),
            "uptime": self.get_timestamp(),
            "components": {
                "brain": "healthy",
                "job_runner": "healthy", 
                "aggregator": "healthy"
            }
        }
    
    def get_timestamp(self) -> str:
        """Get current timestamp as ISO string."""
        return datetime.now().isoformat()
    
    async def process_request_stream(
        self, 
        request_data: Dict[str, Any], 
        request_type: str = "conversation_message",
        correlation_id: Optional[str] = None
    ) -> AsyncGenerator[Dict[str, Any], None]:
        """
        Process request with streaming response.
        
        Args:
            request_data: Request data to process
            request_type: Type of request
            correlation_id: Correlation ID for tracking
        
        Yields:
            Streaming response chunks
        """
        if not correlation_id:
            correlation_id = str(uuid.uuid4())
        
        with with_correlation_id(correlation_id):
            logger.info("Processing streaming request", request_type=request_type)
            
            try:
                # Process through brain with streaming
                async for chunk in self.brain.process_stream(
                    input_data=request_data.get("primary_input", ""),
                    context=request_data.get("context", {}),
                    user_id=request_data.get("user_id", "default"),
                    session_id=request_data.get("session_id", "default"),
                    correlation_id=correlation_id
                ):
                    yield chunk
                    
            except Exception as e:
                logger.error("Streaming request processing failed", error=str(e))
                yield {
                    "type": "error",
                    "error": str(e),
                    "correlation_id": correlation_id
                }
    
    async def process_request(
        self,
        request_data: Dict[str, Any],
        request_type: str = "conversation_message", 
        correlation_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Process a complete request (non-streaming).
        
        Args:
            request_data: Request data to process
            request_type: Type of request
            correlation_id: Correlation ID for tracking
        
        Returns:
            Complete response
        """
        if not correlation_id:
            correlation_id = str(uuid.uuid4())
        
        with with_correlation_id(correlation_id):
            logger.info("Processing request", request_type=request_type)
            
            try:
                # Process through brain
                result = await self.brain.process_input(
                    input_text=request_data.get("primary_input", ""),
                    context=request_data.get("context", {}),
                    user_id=request_data.get("user_id", "default"),
                    session_id=request_data.get("session_id", "default")
                )
                
                return result
                
            except Exception as e:
                logger.error("Request processing failed", error=str(e))
                return {
                    "success": False,
                    "error": str(e),
                    "correlation_id": correlation_id
                }
    
    async def get_session_status(self, session_id: str) -> Dict[str, Any]:
        """Get status of a specific session."""
        # This would interface with memory system to get session info
        try:
            # Get session info from memory system
            session_data = await self.aggregator.memory_system.get_session_data(session_id)
            return {
                "status": "active",
                "last_activity": session_data.get("last_activity"),
                "message_count": session_data.get("message_count", 0),
                "session_id": session_id
            }
        except Exception as e:
            logger.error("Session status retrieval failed", error=str(e))
            return {
                "status": "unknown",
                "error": str(e)
            }
    
    async def reset_session(self, session_id: str) -> bool:
        """Reset a specific session while maintaining continuity."""
        try:
            # Reset through memory system but preserve important context
            await self.aggregator.memory_system.reset_session_context(session_id)
            logger.info("Session reset successfully", session_id=session_id)
            return True
        except Exception as e:
            logger.error("Session reset failed", error=str(e))
            return False
