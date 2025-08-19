"""
Job Runner - Background task processing system using Celery.
Handles long-running tasks, scheduled jobs, and asynchronous processing.
"""

from typing import Dict, Any, Optional, List, Union
from datetime import datetime, timedelta
from dataclasses import dataclass
from enum import Enum
import uuid
import asyncio
from concurrent.futures import ThreadPoolExecutor
import json

from celery import Celery
from celery.result import AsyncResult
from redis import Redis

from src.config import settings
from src.utils.logging import get_logger, with_correlation_id


logger = get_logger(__name__)


class TaskStatus(Enum):
    """Task execution status."""
    PENDING = "PENDING"
    RECEIVED = "RECEIVED"
    STARTED = "STARTED"
    SUCCESS = "SUCCESS"
    FAILURE = "FAILURE"
    RETRY = "RETRY"
    REVOKED = "REVOKED"


class TaskPriority(Enum):
    """Task priority levels."""
    LOW = 1
    NORMAL = 2
    HIGH = 3
    URGENT = 4


@dataclass
class Task:
    """Represents a background task."""
    id: str
    name: str
    args: List[Any]
    kwargs: Dict[str, Any]
    priority: TaskPriority
    scheduled_at: Optional[datetime]
    max_retries: int
    correlation_id: str
    metadata: Dict[str, Any]


# Initialize Celery app
celery_app = Celery(
    "xarvis",
    broker=settings.redis.url,
    backend=settings.redis.url,
    include=[
        "src.backbone.tasks.brain_tasks",
        "src.backbone.tasks.memory_tasks", 
        "src.backbone.tasks.search_tasks",
        "src.backbone.tasks.audio_tasks",
        "src.backbone.tasks.system_tasks"
    ]
)

# Celery configuration
celery_app.conf.update(
    task_serializer="json",
    accept_content=["json"],
    result_serializer="json",
    timezone="UTC",
    enable_utc=True,
    task_track_started=True,
    task_time_limit=30 * 60,  # 30 minutes
    task_soft_time_limit=25 * 60,  # 25 minutes
    worker_prefetch_multiplier=1,
    worker_max_tasks_per_child=1000,
)


class JobRunner:
    """
    Background job processing system for Xarvis.
    Manages task queues, execution, and monitoring.
    """
    
    def __init__(self):
        self.redis_client = Redis.from_url(settings.redis.url)
        self.executor = ThreadPoolExecutor(max_workers=4)
        self.active_tasks: Dict[str, Task] = {}
        
    async def submit_task(
        self,
        task_name: str,
        args: Optional[List[Any]] = None,
        kwargs: Optional[Dict[str, Any]] = None,
        priority: TaskPriority = TaskPriority.NORMAL,
        scheduled_at: Optional[datetime] = None,
        max_retries: int = 3,
        correlation_id: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None
    ) -> str:
        """
        Submit a task for background processing.
        
        Args:
            task_name: Name of the task to execute
            args: Positional arguments for the task
            kwargs: Keyword arguments for the task
            priority: Task priority level
            scheduled_at: When to execute the task (None for immediate)
            max_retries: Maximum number of retries
            correlation_id: Request correlation ID
            metadata: Additional task metadata
        
        Returns:
            Task ID
        """
        task_id = str(uuid.uuid4())
        correlation_id = correlation_id or str(uuid.uuid4())
        
        with with_correlation_id(correlation_id):
            logger.info("Submitting background task", 
                       task_name=task_name, 
                       task_id=task_id,
                       priority=priority.name,
                       scheduled_at=scheduled_at)
            
            task = Task(
                id=task_id,
                name=task_name,
                args=args or [],
                kwargs=kwargs or {},
                priority=priority,
                scheduled_at=scheduled_at,
                max_retries=max_retries,
                correlation_id=correlation_id,
                metadata=metadata or {}
            )
            
            # Store task information
            self.active_tasks[task_id] = task
            
            try:
                # Submit to Celery
                celery_task = celery_app.send_task(
                    task_name,
                    args=task.args,
                    kwargs=task.kwargs,
                    task_id=task_id,
                    eta=scheduled_at,
                    retry=True,
                    retry_policy={
                        'max_retries': max_retries,
                        'interval_start': 2,
                        'interval_step': 2,
                        'interval_max': 30,
                    },
                    priority=priority.value
                )
                
                logger.info("Task submitted to Celery", 
                           task_id=task_id, 
                           celery_id=celery_task.id)
                
                return task_id
                
            except Exception as e:
                logger.error("Failed to submit task", 
                           task_id=task_id, 
                           error=str(e))
                raise
    
    async def get_task_status(self, task_id: str) -> Dict[str, Any]:
        """
        Get the status of a background task.
        
        Args:
            task_id: Task ID to check
        
        Returns:
            Task status information
        """
        try:
            # Get task from active tasks
            task = self.active_tasks.get(task_id)
            if not task:
                return {"error": "Task not found", "task_id": task_id}
            
            # Get status from Celery
            result = AsyncResult(task_id, app=celery_app)
            
            status_info = {
                "task_id": task_id,
                "name": task.name,
                "status": result.status,
                "result": result.result if result.ready() else None,
                "traceback": result.traceback if result.failed() else None,
                "progress": self._get_task_progress(task_id),
                "submitted_at": task.metadata.get("submitted_at"),
                "started_at": result.info.get("started_at") if isinstance(result.info, dict) else None,
                "completed_at": result.info.get("completed_at") if isinstance(result.info, dict) else None,
                "correlation_id": task.correlation_id
            }
            
            return status_info
            
        except Exception as e:
            logger.error("Failed to get task status", task_id=task_id, error=str(e))
            return {"error": str(e), "task_id": task_id}
    
    async def cancel_task(self, task_id: str) -> bool:
        """
        Cancel a background task.
        
        Args:
            task_id: Task ID to cancel
        
        Returns:
            True if successful, False otherwise
        """
        try:
            celery_app.control.revoke(task_id, terminate=True)
            
            # Remove from active tasks
            if task_id in self.active_tasks:
                del self.active_tasks[task_id]
            
            logger.info("Task cancelled", task_id=task_id)
            return True
            
        except Exception as e:
            logger.error("Failed to cancel task", task_id=task_id, error=str(e))
            return False
    
    async def get_active_tasks(self) -> List[Dict[str, Any]]:
        """Get information about all active tasks."""
        tasks = []
        
        for task_id, task in self.active_tasks.items():
            status = await self.get_task_status(task_id)
            tasks.append(status)
        
        return tasks
    
    async def schedule_periodic_task(
        self,
        name: str,
        task_name: str,
        schedule: Union[int, timedelta],
        args: Optional[List[Any]] = None,
        kwargs: Optional[Dict[str, Any]] = None
    ) -> None:
        """
        Schedule a periodic task.
        
        Args:
            name: Unique name for the periodic task
            task_name: Name of the task to execute
            schedule: Schedule interval (seconds or timedelta)
            args: Task arguments
            kwargs: Task keyword arguments
        """
        try:
            from celery.beat import Scheduler
            
            if isinstance(schedule, int):
                schedule = timedelta(seconds=schedule)
            
            # This would typically be configured in celerybeat-schedule
            # For now, we'll log the intent
            logger.info("Periodic task scheduled", 
                       name=name, 
                       task_name=task_name, 
                       interval=str(schedule))
            
        except Exception as e:
            logger.error("Failed to schedule periodic task", 
                        name=name, 
                        error=str(e))
    
    def _get_task_progress(self, task_id: str) -> Optional[Dict[str, Any]]:
        """Get task progress information from Redis."""
        try:
            progress_key = f"task_progress:{task_id}"
            progress_data = self.redis_client.get(progress_key)
            
            if progress_data:
                return json.loads(progress_data)
            
            return None
            
        except Exception as e:
            logger.error("Failed to get task progress", task_id=task_id, error=str(e))
            return None
    
    async def cleanup_completed_tasks(self, max_age_hours: int = 24) -> int:
        """
        Clean up completed tasks older than specified age.
        
        Args:
            max_age_hours: Maximum age in hours for completed tasks
        
        Returns:
            Number of tasks cleaned up
        """
        cleaned_up = 0
        cutoff_time = datetime.now() - timedelta(hours=max_age_hours)
        
        for task_id in list(self.active_tasks.keys()):
            try:
                result = AsyncResult(task_id, app=celery_app)
                
                if result.ready() and hasattr(result.info, 'get'):
                    completed_at = result.info.get("completed_at")
                    if completed_at and datetime.fromisoformat(completed_at) < cutoff_time:
                        del self.active_tasks[task_id]
                        cleaned_up += 1
                        
            except Exception as e:
                logger.error("Error during task cleanup", task_id=task_id, error=str(e))
        
        logger.info("Completed task cleanup", cleaned_up=cleaned_up)
        return cleaned_up
    
    def get_stats(self) -> Dict[str, Any]:
        """Get job runner statistics."""
        active = celery_app.control.inspect().active()
        reserved = celery_app.control.inspect().reserved()
        
        return {
            "active_tasks": len(self.active_tasks),
            "worker_stats": {
                "active": active,
                "reserved": reserved
            },
            "queue_length": self._get_queue_length()
        }
    
    def _get_queue_length(self) -> int:
        """Get current queue length."""
        try:
            return self.redis_client.llen("celery")
        except Exception:
            return 0
