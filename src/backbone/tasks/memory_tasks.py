"""
Memory Tasks - Celery tasks for memory and conversation storage management.
Handles conversation storage, memory retrieval, and vector embeddings.
"""

from typing import Dict, Any, List, Optional
from datetime import datetime, timedelta
import json
import uuid

from celery import current_app
from celery.exceptions import Retry

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


@current_app.task(
    bind=True,
    name="memory.store_conversation",
    max_retries=3,
    default_retry_delay=30
)
def store_conversation(
    self,
    user_id: str,
    conversation_id: str,
    message: str,
    response: str,
    metadata: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Store conversation in memory system.
    
    Args:
        user_id: User identifier
        conversation_id: Conversation identifier
        message: User message
        response: AI response
        metadata: Additional metadata
    
    Returns:
        Storage result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Storing conversation", 
                   user_id=user_id,
                   conversation_id=conversation_id,
                   message_length=len(message),
                   response_length=len(response))
        
        try:
            # Import here to avoid circular imports
            from src.core.memory_system import MemorySystem
            
            memory = MemorySystem()
            
            # Store conversation
            result = memory.store_conversation(
                user_id=user_id,
                conversation_id=conversation_id,
                user_message=message,
                ai_response=response,
                metadata=metadata or {},
                timestamp=datetime.now()
            )
            
            # Generate embeddings for better search (async)
            if result.get("success"):
                update_memory_embeddings.delay(
                    conversation_id=conversation_id,
                    texts=[message, response],
                    metadata={"user_id": user_id, "stored_at": datetime.now().isoformat()}
                )
            
            logger.info("Conversation stored successfully", 
                       user_id=user_id,
                       conversation_id=conversation_id,
                       memory_id=result.get("memory_id"))
            
            return {
                "success": True,
                "result": result,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Conversation storage failed", 
                        user_id=user_id,
                        conversation_id=conversation_id,
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
    name="memory.search_memories",
    max_retries=2,
    default_retry_delay=15
)
def search_memories(
    self,
    query: str,
    user_id: Optional[str] = None,
    limit: int = 10,
    filters: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Search through stored memories.
    
    Args:
        query: Search query
        user_id: Filter by user ID
        limit: Maximum results to return
        filters: Additional search filters
    
    Returns:
        Search results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Searching memories", 
                   query_length=len(query),
                   user_id=user_id,
                   limit=limit)
        
        try:
            # Import here to avoid circular imports
            from src.core.memory_system import MemorySystem
            
            memory = MemorySystem()
            
            # Search memories
            results = memory.search_memories(
                query=query,
                user_id=user_id,
                limit=limit,
                filters=filters or {}
            )
            
            logger.info("Memory search completed", 
                       query=query[:50] + "..." if len(query) > 50 else query,
                       results_count=len(results))
            
            return {
                "success": True,
                "results": results,
                "query": query,
                "count": len(results),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Memory search failed", 
                        query=query,
                        user_id=user_id,
                        error=str(e))
            
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
    name="memory.update_memory_embeddings",
    max_retries=3,
    default_retry_delay=60
)
def update_memory_embeddings(
    self,
    conversation_id: str,
    texts: List[str],
    metadata: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Update vector embeddings for stored memories.
    
    Args:
        conversation_id: Conversation identifier
        texts: Text content to embed
        metadata: Additional metadata
    
    Returns:
        Embedding update result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Updating memory embeddings", 
                   conversation_id=conversation_id,
                   text_count=len(texts))
        
        try:
            # Import here to avoid circular imports
            from src.core.memory_system import MemorySystem
            
            memory = MemorySystem()
            
            # Update embeddings
            result = memory.update_embeddings(
                conversation_id=conversation_id,
                texts=texts,
                metadata=metadata or {}
            )
            
            logger.info("Memory embeddings updated", 
                       conversation_id=conversation_id,
                       embeddings_count=result.get("embeddings_count", 0))
            
            return {
                "success": True,
                "result": result,
                "conversation_id": conversation_id,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Memory embeddings update failed", 
                        conversation_id=conversation_id,
                        error=str(e))
            
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
    name="memory.cleanup_old_memories",
    max_retries=2
)
def cleanup_old_memories(
    self,
    older_than_days: int = 90,
    keep_important: bool = True,
    dry_run: bool = False
) -> Dict[str, Any]:
    """
    Clean up old memories to manage storage.
    
    Args:
        older_than_days: Remove memories older than this many days
        keep_important: Keep memories marked as important
        dry_run: Don't actually delete, just report what would be deleted
    
    Returns:
        Cleanup results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Cleaning up old memories", 
                   older_than_days=older_than_days,
                   keep_important=keep_important,
                   dry_run=dry_run)
        
        try:
            # Import here to avoid circular imports
            from src.core.memory_system import MemorySystem
            
            memory = MemorySystem()
            
            # Calculate cutoff date
            cutoff_date = datetime.now() - timedelta(days=older_than_days)
            
            # Clean up memories
            result = memory.cleanup_old_memories(
                cutoff_date=cutoff_date,
                keep_important=keep_important,
                dry_run=dry_run
            )
            
            logger.info("Memory cleanup completed", 
                       removed_count=result.get("removed_count", 0),
                       kept_count=result.get("kept_count", 0),
                       dry_run=dry_run)
            
            return {
                "success": True,
                "result": result,
                "cutoff_date": cutoff_date.isoformat(),
                "dry_run": dry_run,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Memory cleanup failed", error=str(e))
            
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
    name="memory.export_user_memories",
    max_retries=2
)
def export_user_memories(
    self,
    user_id: str,
    format: str = "json",
    include_metadata: bool = True
) -> Dict[str, Any]:
    """
    Export user's memories for backup or migration.
    
    Args:
        user_id: User identifier
        format: Export format (json, csv)
        include_metadata: Include metadata in export
    
    Returns:
        Export result
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Exporting user memories", 
                   user_id=user_id,
                   format=format,
                   include_metadata=include_metadata)
        
        try:
            # Import here to avoid circular imports
            from src.core.memory_system import MemorySystem
            
            memory = MemorySystem()
            
            # Export memories
            export_data = memory.export_user_memories(
                user_id=user_id,
                format=format,
                include_metadata=include_metadata
            )
            
            logger.info("User memories exported", 
                       user_id=user_id,
                       format=format,
                       data_size=len(str(export_data)))
            
            return {
                "success": True,
                "export_data": export_data,
                "user_id": user_id,
                "format": format,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Memory export failed", 
                        user_id=user_id,
                        error=str(e))
            
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
    name="memory.rebuild_embeddings",
    max_retries=2
)
def rebuild_embeddings(
    self,
    user_id: Optional[str] = None,
    batch_size: int = 100
) -> Dict[str, Any]:
    """
    Rebuild vector embeddings for memories.
    
    Args:
        user_id: Rebuild for specific user (None for all users)
        batch_size: Number of memories to process at once
    
    Returns:
        Rebuild results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Rebuilding memory embeddings", 
                   user_id=user_id,
                   batch_size=batch_size)
        
        try:
            # Import here to avoid circular imports
            from src.core.memory_system import MemorySystem
            
            memory = MemorySystem()
            
            # Rebuild embeddings
            result = memory.rebuild_embeddings(
                user_id=user_id,
                batch_size=batch_size
            )
            
            logger.info("Embeddings rebuild completed", 
                       processed_count=result.get("processed_count", 0),
                       user_id=user_id)
            
            return {
                "success": True,
                "result": result,
                "user_id": user_id,
                "batch_size": batch_size,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Embeddings rebuild failed", 
                        user_id=user_id,
                        error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=120, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="memory.memory_statistics",
    max_retries=1
)
def memory_statistics(
    self,
    user_id: Optional[str] = None,
    time_range_days: int = 30
) -> Dict[str, Any]:
    """
    Generate memory usage statistics.
    
    Args:
        user_id: Generate stats for specific user
        time_range_days: Time range for statistics
    
    Returns:
        Memory statistics
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Generating memory statistics", 
                   user_id=user_id,
                   time_range_days=time_range_days)
        
        try:
            # Import here to avoid circular imports
            from src.core.memory_system import MemorySystem
            
            memory = MemorySystem()
            
            # Generate statistics
            stats = memory.get_statistics(
                user_id=user_id,
                time_range_days=time_range_days
            )
            
            logger.info("Memory statistics generated", 
                       total_conversations=stats.get("total_conversations", 0),
                       total_memories=stats.get("total_memories", 0),
                       user_id=user_id)
            
            return {
                "success": True,
                "statistics": stats,
                "user_id": user_id,
                "time_range_days": time_range_days,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Memory statistics generation failed", 
                        user_id=user_id,
                        error=str(e))
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


# Export tasks
__all__ = [
    "store_conversation",
    "search_memories",
    "update_memory_embeddings",
    "cleanup_old_memories",
    "export_user_memories",
    "rebuild_embeddings",
    "memory_statistics"
]
