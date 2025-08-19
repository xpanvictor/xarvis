"""
Database Models - SQLAlchemy models for Jarvis system.
Defines database schema for conversations, memories, users, and system data.
"""

from typing import Dict, Any, List, Optional
from datetime import datetime
from sqlalchemy import (
    Column, Integer, String, Text, DateTime, Boolean, Float, JSON,
    ForeignKey, Index, UniqueConstraint, CheckConstraint
)
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy.orm import relationship, backref
from sqlalchemy.dialects.postgresql import UUID, ARRAY
import uuid

from src.utils.logging import get_logger


logger = get_logger(__name__)

# Create base class for all models
Base = declarative_base()


class User(Base):
    """User model for storing user information."""
    __tablename__ = "users"
    
    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    username = Column(String(100), unique=True, nullable=False)
    email = Column(String(255), unique=True, nullable=True)
    full_name = Column(String(255), nullable=True)
    
    # User preferences and settings
    preferences = Column(JSON, default=dict)
    settings = Column(JSON, default=dict)
    
    # Status and metadata
    is_active = Column(Boolean, default=True, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False)
    last_seen = Column(DateTime, nullable=True)
    
    # Relationships
    conversations = relationship("Conversation", back_populates="user", cascade="all, delete-orphan")
    memories = relationship("Memory", back_populates="user", cascade="all, delete-orphan")
    
    # Indexes
    __table_args__ = (
        Index("ix_users_username", username),
        Index("ix_users_email", email),
        Index("ix_users_created_at", created_at),
    )
    
    def __repr__(self):
        return f"<User(id='{self.id}', username='{self.username}')>"
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "username": self.username,
            "email": self.email,
            "full_name": self.full_name,
            "preferences": self.preferences,
            "settings": self.settings,
            "is_active": self.is_active,
            "created_at": self.created_at.isoformat() if self.created_at else None,
            "updated_at": self.updated_at.isoformat() if self.updated_at else None,
            "last_seen": self.last_seen.isoformat() if self.last_seen else None
        }


class Conversation(Base):
    """Conversation model for storing conversation sessions."""
    __tablename__ = "conversations"
    
    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    user_id = Column(String(36), ForeignKey("users.id"), nullable=False)
    
    # Conversation metadata
    title = Column(String(255), nullable=True)
    context = Column(JSON, default=dict)
    
    # Status and timing
    status = Column(String(50), default="active", nullable=False)  # active, completed, archived
    started_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    ended_at = Column(DateTime, nullable=True)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False)
    
    # Conversation statistics
    message_count = Column(Integer, default=0, nullable=False)
    total_tokens = Column(Integer, default=0, nullable=False)
    
    # Relationships
    user = relationship("User", back_populates="conversations")
    messages = relationship("Message", back_populates="conversation", cascade="all, delete-orphan")
    memories = relationship("Memory", back_populates="conversation")
    
    # Indexes
    __table_args__ = (
        Index("ix_conversations_user_id", user_id),
        Index("ix_conversations_started_at", started_at),
        Index("ix_conversations_status", status),
        CheckConstraint("status IN ('active', 'completed', 'archived')", name="check_conversation_status")
    )
    
    def __repr__(self):
        return f"<Conversation(id='{self.id}', user_id='{self.user_id}', status='{self.status}')>"
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "user_id": self.user_id,
            "title": self.title,
            "context": self.context,
            "status": self.status,
            "started_at": self.started_at.isoformat() if self.started_at else None,
            "ended_at": self.ended_at.isoformat() if self.ended_at else None,
            "updated_at": self.updated_at.isoformat() if self.updated_at else None,
            "message_count": self.message_count,
            "total_tokens": self.total_tokens
        }


class Message(Base):
    """Message model for storing individual messages in conversations."""
    __tablename__ = "messages"
    
    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    conversation_id = Column(String(36), ForeignKey("conversations.id"), nullable=False)
    
    # Message content
    role = Column(String(20), nullable=False)  # user, assistant, system
    content = Column(Text, nullable=False)
    
    # Message metadata
    metadata = Column(JSON, default=dict)
    tokens_used = Column(Integer, default=0, nullable=False)
    processing_time = Column(Float, default=0.0, nullable=False)
    
    # Tool usage
    tools_used = Column(JSON, default=list)  # List of tools used in this message
    tool_results = Column(JSON, default=dict)  # Results from tool calls
    
    # Timing
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    
    # Relationships
    conversation = relationship("Conversation", back_populates="messages")
    
    # Indexes
    __table_args__ = (
        Index("ix_messages_conversation_id", conversation_id),
        Index("ix_messages_created_at", created_at),
        Index("ix_messages_role", role),
        CheckConstraint("role IN ('user', 'assistant', 'system')", name="check_message_role")
    )
    
    def __repr__(self):
        return f"<Message(id='{self.id}', conversation_id='{self.conversation_id}', role='{self.role}')>"
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "conversation_id": self.conversation_id,
            "role": self.role,
            "content": self.content,
            "metadata": self.metadata,
            "tokens_used": self.tokens_used,
            "processing_time": self.processing_time,
            "tools_used": self.tools_used,
            "tool_results": self.tool_results,
            "created_at": self.created_at.isoformat() if self.created_at else None
        }


class Memory(Base):
    """Memory model for storing long-term memories and knowledge."""
    __tablename__ = "memories"
    
    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    user_id = Column(String(36), ForeignKey("users.id"), nullable=False)
    conversation_id = Column(String(36), ForeignKey("conversations.id"), nullable=True)
    
    # Memory content
    content = Column(Text, nullable=False)
    summary = Column(Text, nullable=True)
    
    # Memory classification
    memory_type = Column(String(50), default="conversation", nullable=False)  # conversation, fact, skill, preference
    importance_score = Column(Float, default=0.5, nullable=False)
    
    # Vector embeddings (stored as JSON for PostgreSQL compatibility)
    embedding = Column(JSON, nullable=True)
    embedding_model = Column(String(100), nullable=True)
    
    # Memory metadata
    metadata = Column(JSON, default=dict)
    tags = Column(JSON, default=list)  # List of tags for categorization
    
    # Status and timing
    is_active = Column(Boolean, default=True, nullable=False)
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False)
    last_accessed = Column(DateTime, nullable=True)
    access_count = Column(Integer, default=0, nullable=False)
    
    # Relationships
    user = relationship("User", back_populates="memories")
    conversation = relationship("Conversation", back_populates="memories")
    
    # Indexes
    __table_args__ = (
        Index("ix_memories_user_id", user_id),
        Index("ix_memories_conversation_id", conversation_id),
        Index("ix_memories_memory_type", memory_type),
        Index("ix_memories_importance_score", importance_score),
        Index("ix_memories_created_at", created_at),
        Index("ix_memories_is_active", is_active),
        CheckConstraint("memory_type IN ('conversation', 'fact', 'skill', 'preference')", name="check_memory_type"),
        CheckConstraint("importance_score >= 0.0 AND importance_score <= 1.0", name="check_importance_score")
    )
    
    def __repr__(self):
        return f"<Memory(id='{self.id}', user_id='{self.user_id}', type='{self.memory_type}')>"
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "user_id": self.user_id,
            "conversation_id": self.conversation_id,
            "content": self.content,
            "summary": self.summary,
            "memory_type": self.memory_type,
            "importance_score": self.importance_score,
            "metadata": self.metadata,
            "tags": self.tags,
            "is_active": self.is_active,
            "created_at": self.created_at.isoformat() if self.created_at else None,
            "updated_at": self.updated_at.isoformat() if self.updated_at else None,
            "last_accessed": self.last_accessed.isoformat() if self.last_accessed else None,
            "access_count": self.access_count
        }


class Tool(Base):
    """Tool model for storing tool definitions and usage statistics."""
    __tablename__ = "tools"
    
    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    
    # Tool definition
    name = Column(String(100), unique=True, nullable=False)
    description = Column(Text, nullable=False)
    
    # Tool specification
    schema = Column(JSON, nullable=False)  # JSON schema for tool parameters
    category = Column(String(50), default="general", nullable=False)
    
    # Tool status and metadata
    is_active = Column(Boolean, default=True, nullable=False)
    is_builtin = Column(Boolean, default=False, nullable=False)
    version = Column(String(20), default="1.0", nullable=False)
    
    # Usage statistics
    usage_count = Column(Integer, default=0, nullable=False)
    success_count = Column(Integer, default=0, nullable=False)
    error_count = Column(Integer, default=0, nullable=False)
    average_execution_time = Column(Float, default=0.0, nullable=False)
    
    # Timing
    created_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False)
    last_used = Column(DateTime, nullable=True)
    
    # Indexes
    __table_args__ = (
        Index("ix_tools_name", name),
        Index("ix_tools_category", category),
        Index("ix_tools_is_active", is_active),
        Index("ix_tools_usage_count", usage_count),
    )
    
    def __repr__(self):
        return f"<Tool(id='{self.id}', name='{self.name}', category='{self.category}')>"
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "name": self.name,
            "description": self.description,
            "schema": self.schema,
            "category": self.category,
            "is_active": self.is_active,
            "is_builtin": self.is_builtin,
            "version": self.version,
            "usage_count": self.usage_count,
            "success_count": self.success_count,
            "error_count": self.error_count,
            "average_execution_time": self.average_execution_time,
            "created_at": self.created_at.isoformat() if self.created_at else None,
            "updated_at": self.updated_at.isoformat() if self.updated_at else None,
            "last_used": self.last_used.isoformat() if self.last_used else None
        }


class SystemLog(Base):
    """System log model for storing application logs and events."""
    __tablename__ = "system_logs"
    
    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    
    # Log content
    level = Column(String(20), nullable=False)  # DEBUG, INFO, WARNING, ERROR, CRITICAL
    message = Column(Text, nullable=False)
    logger_name = Column(String(100), nullable=False)
    
    # Context
    correlation_id = Column(String(36), nullable=True)
    user_id = Column(String(36), nullable=True)
    
    # Log metadata
    metadata = Column(JSON, default=dict)
    
    # Timing
    timestamp = Column(DateTime, default=datetime.utcnow, nullable=False)
    
    # Indexes
    __table_args__ = (
        Index("ix_system_logs_level", level),
        Index("ix_system_logs_timestamp", timestamp),
        Index("ix_system_logs_correlation_id", correlation_id),
        Index("ix_system_logs_user_id", user_id),
        Index("ix_system_logs_logger_name", logger_name),
        CheckConstraint("level IN ('DEBUG', 'INFO', 'WARNING', 'ERROR', 'CRITICAL')", name="check_log_level")
    )
    
    def __repr__(self):
        return f"<SystemLog(id='{self.id}', level='{self.level}', logger='{self.logger_name}')>"
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "level": self.level,
            "message": self.message,
            "logger_name": self.logger_name,
            "correlation_id": self.correlation_id,
            "user_id": self.user_id,
            "metadata": self.metadata,
            "timestamp": self.timestamp.isoformat() if self.timestamp else None
        }


class AudioFile(Base):
    """Audio file model for storing audio processing information."""
    __tablename__ = "audio_files"
    
    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    user_id = Column(String(36), ForeignKey("users.id"), nullable=True)
    
    # File information
    filename = Column(String(255), nullable=False)
    file_path = Column(String(500), nullable=False)
    file_size = Column(Integer, nullable=False)
    
    # Audio properties
    format = Column(String(20), nullable=False)  # wav, mp3, etc.
    duration = Column(Float, nullable=True)
    sample_rate = Column(Integer, nullable=True)
    channels = Column(Integer, nullable=True)
    
    # Processing information
    transcription = Column(Text, nullable=True)
    transcription_model = Column(String(50), nullable=True)
    transcription_confidence = Column(Float, nullable=True)
    
    # Audio metadata
    metadata = Column(JSON, default=dict)
    
    # Status
    processing_status = Column(String(50), default="uploaded", nullable=False)  # uploaded, processing, completed, error
    
    # Timing
    uploaded_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    processed_at = Column(DateTime, nullable=True)
    
    # Relationships
    user = relationship("User")
    
    # Indexes
    __table_args__ = (
        Index("ix_audio_files_user_id", user_id),
        Index("ix_audio_files_uploaded_at", uploaded_at),
        Index("ix_audio_files_processing_status", processing_status),
        Index("ix_audio_files_format", format),
        CheckConstraint("processing_status IN ('uploaded', 'processing', 'completed', 'error')", name="check_processing_status")
    )
    
    def __repr__(self):
        return f"<AudioFile(id='{self.id}', filename='{self.filename}', status='{self.processing_status}')>"
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "user_id": self.user_id,
            "filename": self.filename,
            "file_path": self.file_path,
            "file_size": self.file_size,
            "format": self.format,
            "duration": self.duration,
            "sample_rate": self.sample_rate,
            "channels": self.channels,
            "transcription": self.transcription,
            "transcription_model": self.transcription_model,
            "transcription_confidence": self.transcription_confidence,
            "metadata": self.metadata,
            "processing_status": self.processing_status,
            "uploaded_at": self.uploaded_at.isoformat() if self.uploaded_at else None,
            "processed_at": self.processed_at.isoformat() if self.processed_at else None
        }


class HardwareDevice(Base):
    """Hardware device model for storing device information and status."""
    __tablename__ = "hardware_devices"
    
    id = Column(String(36), primary_key=True, default=lambda: str(uuid.uuid4()))
    
    # Device identification
    device_id = Column(String(100), unique=True, nullable=False)  # External device ID
    name = Column(String(100), nullable=False)
    device_type = Column(String(50), nullable=False)  # esp32, arduino, sensor, etc.
    
    # Connection information
    connection_type = Column(String(50), nullable=False)  # serial, mqtt, http, etc.
    address = Column(String(255), nullable=False)  # Port, IP, URL, etc.
    
    # Device capabilities and configuration
    capabilities = Column(JSON, default=list)  # List of device capabilities
    configuration = Column(JSON, default=dict)  # Device configuration
    
    # Status information
    status = Column(String(50), default="disconnected", nullable=False)  # connected, disconnected, error, etc.
    last_seen = Column(DateTime, nullable=True)
    
    # Statistics
    uptime = Column(Integer, default=0, nullable=False)  # Seconds
    message_count = Column(Integer, default=0, nullable=False)
    error_count = Column(Integer, default=0, nullable=False)
    
    # Device metadata
    metadata = Column(JSON, default=dict)
    firmware_version = Column(String(50), nullable=True)
    
    # Timing
    registered_at = Column(DateTime, default=datetime.utcnow, nullable=False)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow, nullable=False)
    
    # Indexes
    __table_args__ = (
        Index("ix_hardware_devices_device_id", device_id),
        Index("ix_hardware_devices_device_type", device_type),
        Index("ix_hardware_devices_status", status),
        Index("ix_hardware_devices_last_seen", last_seen),
        CheckConstraint("status IN ('connected', 'disconnected', 'error', 'busy', 'idle')", name="check_device_status")
    )
    
    def __repr__(self):
        return f"<HardwareDevice(id='{self.id}', device_id='{self.device_id}', type='{self.device_type}')>"
    
    def to_dict(self) -> Dict[str, Any]:
        return {
            "id": self.id,
            "device_id": self.device_id,
            "name": self.name,
            "device_type": self.device_type,
            "connection_type": self.connection_type,
            "address": self.address,
            "capabilities": self.capabilities,
            "configuration": self.configuration,
            "status": self.status,
            "last_seen": self.last_seen.isoformat() if self.last_seen else None,
            "uptime": self.uptime,
            "message_count": self.message_count,
            "error_count": self.error_count,
            "metadata": self.metadata,
            "firmware_version": self.firmware_version,
            "registered_at": self.registered_at.isoformat() if self.registered_at else None,
            "updated_at": self.updated_at.isoformat() if self.updated_at else None
        }


# Export all models
__all__ = [
    "Base",
    "User",
    "Conversation",
    "Message", 
    "Memory",
    "Tool",
    "SystemLog",
    "AudioFile",
    "HardwareDevice"
]
