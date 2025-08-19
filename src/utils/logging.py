"""
Logging utilities for Xarvis system.
Provides structured logging with context and correlation IDs.
"""

import logging
import sys
import json
from datetime import datetime
from typing import Any, Dict, Optional
from pathlib import Path
import structlog
from structlog.stdlib import LoggerFactory
import uuid


class XarvisProcessor:
    """Custom processor for adding Xarvis-specific context to logs."""
    
    def __call__(self, logger: Any, method_name: str, event_dict: Dict[str, Any]) -> Dict[str, Any]:
        """Add Xarvis-specific context to log entries."""
        event_dict["service"] = "xarvis"
        event_dict["timestamp"] = datetime.utcnow().isoformat()
        
        # Add correlation ID if not present
        if "correlation_id" not in event_dict:
            event_dict["correlation_id"] = str(uuid.uuid4())
        
        return event_dict


def setup_logging(
    log_level: str = "INFO",
    log_format: str = "json",
    log_file: Optional[str] = None
) -> None:
    """
    Setup structured logging for the application.
    
    Args:
        log_level: Logging level (DEBUG, INFO, WARNING, ERROR, CRITICAL)
        log_format: Format type ("json" or "console")
        log_file: Path to log file (optional)
    """
    # Ensure logs directory exists
    if log_file:
        Path(log_file).parent.mkdir(parents=True, exist_ok=True)
    
    # Configure structlog
    processors = [
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        XarvisProcessor(),
        structlog.processors.format_exc_info,
    ]
    
    if log_format == "json":
        processors.append(structlog.processors.JSONRenderer())
    else:
        processors.append(structlog.dev.ConsoleRenderer())
    
    structlog.configure(
        processors=processors,
        wrapper_class=structlog.stdlib.BoundLogger,
        logger_factory=LoggerFactory(),
        wrapper_factory=structlog.stdlib.BoundLogger,
        cache_logger_on_first_use=True,
    )
    
    # Configure standard logging
    handlers = []
    
    # Console handler
    console_handler = logging.StreamHandler(sys.stdout)
    console_handler.setLevel(getattr(logging, log_level.upper()))
    handlers.append(console_handler)
    
    # File handler if specified
    if log_file:
        file_handler = logging.FileHandler(log_file)
        file_handler.setLevel(getattr(logging, log_level.upper()))
        handlers.append(file_handler)
    
    # Configure root logger
    logging.basicConfig(
        level=getattr(logging, log_level.upper()),
        handlers=handlers,
        format="%(message)s"
    )


def get_logger(name: str) -> structlog.BoundLogger:
    """
    Get a structured logger instance.
    
    Args:
        name: Logger name (usually __name__)
    
    Returns:
        Configured structured logger
    """
    return structlog.get_logger(name)


class LogContext:
    """Context manager for adding contextual information to logs."""
    
    def __init__(self, **context: Any):
        self.context = context
        self.token = None
    
    def __enter__(self) -> "LogContext":
        self.token = structlog.contextvars.bind_contextvars(**self.context)
        return self
    
    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if self.token:
            structlog.contextvars.reset_contextvars(self.token)


def with_correlation_id(correlation_id: Optional[str] = None) -> LogContext:
    """
    Context manager for adding correlation ID to logs.
    
    Args:
        correlation_id: Correlation ID (generates one if None)
    
    Returns:
        LogContext with correlation ID
    """
    if correlation_id is None:
        correlation_id = str(uuid.uuid4())
    
    return LogContext(correlation_id=correlation_id)


# Default logger instance
logger = get_logger(__name__)
