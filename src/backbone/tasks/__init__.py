"""
Celery tasks package - Background processing for Jarvis system.
Provides distributed task processing for AI operations, memory management, and hardware interactions.
"""

from .brain_tasks import *
from .memory_tasks import *
from .search_tasks import *
from .audio_tasks import *
from .system_tasks import *

__all__ = [
    # Brain tasks
    "process_conversation",
    "generate_response",
    "analyze_intent",
    "execute_tool_call",
    
    # Memory tasks
    "store_conversation",
    "search_memories",
    "update_memory_embeddings",
    "cleanup_old_memories",
    
    # Search tasks
    "web_search",
    "search_documentation",
    "search_code_repositories",
    
    # Audio tasks
    "process_speech_to_text",
    "generate_text_to_speech",
    "process_audio_file",
    
    # System tasks
    "health_check",
    "system_monitoring",
    "cleanup_tasks",
    "backup_data"
]
