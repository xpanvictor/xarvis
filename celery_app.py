"""
Celery Worker Configuration - Celery app configuration and task discovery.
Configures Celery with Redis broker and task routing.
"""

from celery import Celery
from src.config import settings
from src.utils.logging import get_logger

logger = get_logger(__name__)

# Create Celery app
app = Celery(
    'xarvis',
    broker=settings.redis.url,
    backend=settings.redis.url,
    include=[
        'src.backbone.tasks.brain_tasks',
        'src.backbone.tasks.memory_tasks', 
        'src.backbone.tasks.search_tasks',
        'src.backbone.tasks.audio_tasks',
        'src.backbone.tasks.system_tasks'
    ]
)

# Celery configuration
app.conf.update(
    # Task settings
    task_serializer='json',
    accept_content=['json'],
    result_serializer='json',
    timezone='UTC',
    enable_utc=True,
    
    # Result backend settings
    result_expires=3600,  # 1 hour
    result_backend_transport_options={
        'master_name': 'mymaster',
        'visibility_timeout': 3600,
    },
    
    # Worker settings
    worker_prefetch_multiplier=1,
    worker_max_tasks_per_child=1000,
    worker_disable_rate_limits=False,
    
    # Task routing
    task_routes={
        'brain.*': {'queue': 'brain'},
        'memory.*': {'queue': 'memory'},
        'search.*': {'queue': 'search'},
        'audio.*': {'queue': 'audio'},
        'system.*': {'queue': 'system'},
    },
    
    # Task execution settings
    task_always_eager=False,
    task_eager_propagates=True,
    task_ignore_result=False,
    task_store_eager_result=True,
    
    # Beat settings (for periodic tasks)
    beat_schedule={
        'system-health-check': {
            'task': 'system.health_check',
            'schedule': 300.0,  # Every 5 minutes
        },
        'cleanup-audio-cache': {
            'task': 'audio.cleanup_audio_cache',
            'schedule': 3600.0,  # Every hour
        },
        'cleanup-old-memories': {
            'task': 'memory.cleanup_old_memories',
            'schedule': 86400.0,  # Every day
            'kwargs': {'older_than_days': 90}
        },
        'system-monitoring': {
            'task': 'system.system_monitoring',
            'schedule': 60.0,  # Every minute
        },
    },
    beat_schedule_filename='celerybeat-schedule',
    
    # Redis specific settings
    broker_connection_retry_on_startup=True,
    broker_connection_retry=True,
    broker_connection_max_retries=10,
)

# Task discovery
app.autodiscover_tasks([
    'src.backbone.tasks'
])

logger.info("Celery app configured successfully")

if __name__ == '__main__':
    app.start()
