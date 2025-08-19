"""
Brain Tasks - Celery tasks for AI processing and conversation management.
Handles conversation processing, response generation, and tool execution.
"""

from typing import Dict, Any, List, Optional, Union
from datetime import datetime
import json
import uuid

from celery import current_app
from celery.exceptions import Retry

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


@current_app.task(
    bind=True,
    name="brain.process_conversation",
    max_retries=3,
    default_retry_delay=60
)
def process_conversation(
    self,
    message: str,
    user_id: str,
    conversation_id: str,
    context: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Process a complete conversation interaction.
    
    Args:
        message: User input message
        user_id: User identifier
        conversation_id: Conversation identifier
        context: Additional context
    
    Returns:
        Conversation processing result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Processing conversation", 
                   user_id=user_id,
                   conversation_id=conversation_id,
                   message_length=len(message))
        
        try:
            # Import here to avoid circular imports
            from src.backbone.brain import Brain
            from src.core.memory_system import MemorySystem
            
            # Initialize brain
            brain = Brain()
            
            # Process the conversation
            result = brain.process_conversation(
                message=message,
                user_id=user_id,
                conversation_id=conversation_id,
                context=context or {}
            )
            
            # Store in memory (async task)
            store_conversation.delay(
                user_id=user_id,
                conversation_id=conversation_id,
                message=message,
                response=result.get("response", ""),
                metadata={
                    "processing_time": result.get("processing_time", 0),
                    "tools_used": result.get("tools_used", []),
                    "correlation_id": correlation_id
                }
            )
            
            logger.info("Conversation processed successfully", 
                       user_id=user_id,
                       conversation_id=conversation_id,
                       tools_used=result.get("tools_used", []))
            
            return {
                "success": True,
                "result": result,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Conversation processing failed", 
                        user_id=user_id,
                        conversation_id=conversation_id,
                        error=str(e))
            
            # Retry with exponential backoff
            if self.request.retries < self.max_retries:
                raise self.retry(
                    countdown=2 ** self.request.retries,
                    exc=e
                )
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="brain.generate_response",
    max_retries=2,
    default_retry_delay=30
)
def generate_response(
    self,
    message: str,
    context: Dict[str, Any],
    model: str = "gpt-4"
) -> Dict[str, Any]:
    """
    Generate AI response for a message.
    
    Args:
        message: Input message
        context: Conversation context
        model: AI model to use
    
    Returns:
        Generated response
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Generating AI response", 
                   message_length=len(message),
                   model=model)
        
        try:
            # Import here to avoid circular imports
            from src.backbone.brain import Brain
            
            brain = Brain()
            
            # Generate response
            response = brain.generate_response(
                message=message,
                context=context,
                model=model
            )
            
            logger.info("Response generated successfully", 
                       response_length=len(response.get("text", "")),
                       model=model)
            
            return {
                "success": True,
                "response": response,
                "model": model,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Response generation failed", 
                        model=model,
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
    name="brain.analyze_intent",
    max_retries=2,
    default_retry_delay=15
)
def analyze_intent(
    self,
    message: str,
    user_context: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Analyze intent and entities in user message.
    
    Args:
        message: User input message
        user_context: User context information
    
    Returns:
        Intent analysis result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Analyzing intent", message_length=len(message))
        
        try:
            # Import here to avoid circular imports
            from src.core.nlp_processor import NLPProcessor
            
            nlp = NLPProcessor()
            
            # Analyze intent
            analysis = nlp.analyze_intent(message, user_context or {})
            
            logger.info("Intent analyzed successfully", 
                       intent=analysis.get("intent"),
                       confidence=analysis.get("confidence", 0))
            
            return {
                "success": True,
                "analysis": analysis,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Intent analysis failed", error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=15, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="brain.execute_tool_call",
    max_retries=3,
    default_retry_delay=30
)
def execute_tool_call(
    self,
    tool_name: str,
    parameters: Dict[str, Any],
    user_id: str,
    context: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Execute a tool call with given parameters.
    
    Args:
        tool_name: Name of tool to execute
        parameters: Tool parameters
        user_id: User identifier
        context: Execution context
    
    Returns:
        Tool execution result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Executing tool call", 
                   tool_name=tool_name,
                   user_id=user_id,
                   parameters=list(parameters.keys()))
        
        try:
            # Import here to avoid circular imports
            from src.core.tool_registry import ToolRegistry
            
            registry = ToolRegistry()
            
            # Execute tool
            result = registry.execute_tool(
                tool_name=tool_name,
                parameters=parameters,
                user_id=user_id,
                context=context or {}
            )
            
            logger.info("Tool executed successfully", 
                       tool_name=tool_name,
                       user_id=user_id,
                       success=result.get("success", False))
            
            return {
                "success": True,
                "result": result,
                "tool_name": tool_name,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Tool execution failed", 
                        tool_name=tool_name,
                        user_id=user_id,
                        error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=30, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "tool_name": tool_name,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="brain.batch_process_messages",
    max_retries=2
)
def batch_process_messages(
    self,
    messages: List[Dict[str, Any]],
    processing_options: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Process multiple messages in batch.
    
    Args:
        messages: List of messages to process
        processing_options: Batch processing options
    
    Returns:
        Batch processing results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Processing message batch", count=len(messages))
        
        try:
            results = []
            errors = []
            
            for i, message_data in enumerate(messages):
                try:
                    # Process individual message
                    result = process_conversation.apply_async(
                        args=[
                            message_data.get("message", ""),
                            message_data.get("user_id", ""),
                            message_data.get("conversation_id", f"batch_{correlation_id}_{i}"),
                            message_data.get("context")
                        ]
                    ).get(timeout=60)
                    
                    results.append({
                        "index": i,
                        "success": True,
                        "result": result
                    })
                    
                except Exception as e:
                    errors.append({
                        "index": i,
                        "error": str(e),
                        "message_id": message_data.get("id", f"message_{i}")
                    })
            
            logger.info("Batch processing completed", 
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
            logger.error("Batch processing failed", error=str(e))
            
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
    name="brain.context_analysis",
    max_retries=2
)
def context_analysis(
    self,
    conversation_history: List[Dict[str, Any]],
    current_message: str,
    user_profile: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Analyze conversation context and user patterns.
    
    Args:
        conversation_history: Previous conversation messages
        current_message: Current user message
        user_profile: User profile information
    
    Returns:
        Context analysis results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Analyzing conversation context", 
                   history_length=len(conversation_history),
                   message_length=len(current_message))
        
        try:
            # Import here to avoid circular imports
            from src.core.nlp_processor import NLPProcessor
            
            nlp = NLPProcessor()
            
            # Analyze context
            analysis = nlp.analyze_context(
                conversation_history=conversation_history,
                current_message=current_message,
                user_profile=user_profile or {}
            )
            
            logger.info("Context analysis completed", 
                       sentiment=analysis.get("sentiment"),
                       topics=len(analysis.get("topics", [])))
            
            return {
                "success": True,
                "analysis": analysis,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Context analysis failed", error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=30, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


# Import at the end to avoid circular imports
try:
    from .memory_tasks import store_conversation
except ImportError:
    # Handle case where memory_tasks isn't loaded yet
    def store_conversation(*args, **kwargs):
        logger.warning("Memory tasks not available for conversation storage")


# Export tasks
__all__ = [
    "process_conversation",
    "generate_response",
    "analyze_intent",
    "execute_tool_call",
    "batch_process_messages",
    "context_analysis"
]
