"""
Configuration management for Xarvis system.
Handles environment variables and system settings.
"""

from typing import Optional, List
from pydantic_settings import BaseSettings
from pydantic import Field
import os


class APISettings(BaseSettings):
    """API server configuration."""
    
    host: str = Field(default="0.0.0.0", env="API_HOST")
    port: int = Field(default=8000, env="API_PORT")
    workers: int = Field(default=4, env="API_WORKERS")
    reload: bool = Field(default=False, env="API_RELOAD")


class AISettings(BaseSettings):
    """AI service configuration."""
    
    openai_api_key: Optional[str] = Field(default=None, env="OPENAI_API_KEY")
    openai_model: str = Field(default="gpt-4-turbo-preview", env="OPENAI_MODEL")
    anthropic_api_key: Optional[str] = Field(default=None, env="ANTHROPIC_API_KEY")
    anthropic_model: str = Field(default="claude-3-sonnet-20240229", env="ANTHROPIC_MODEL")


class DatabaseSettings(BaseSettings):
    """Database configuration."""
    
    url: str = Field(default="postgresql://xarvis:xarvis_password@localhost:5432/xarvis", env="DATABASE_URL")
    echo: bool = Field(default=False, env="DATABASE_ECHO")
    pool_size: int = Field(default=10, env="DATABASE_POOL_SIZE")


class RedisSettings(BaseSettings):
    """Redis configuration."""
    
    url: str = Field(default="redis://localhost:6379", env="REDIS_URL")
    db: int = Field(default=0, env="REDIS_DB")


class AudioSettings(BaseSettings):
    """Audio processing configuration."""
    
    input_device: Optional[int] = Field(default=None, env="AUDIO_INPUT_DEVICE")
    output_device: Optional[int] = Field(default=None, env="AUDIO_OUTPUT_DEVICE")
    sample_rate: int = Field(default=44100, env="AUDIO_SAMPLE_RATE")
    channels: int = Field(default=1, env="AUDIO_CHANNELS")
    whisper_model: str = Field(default="base", env="WHISPER_MODEL")


class HardwareSettings(BaseSettings):
    """Hardware communication configuration."""
    
    esp32_serial_port: str = Field(default="/dev/ttyUSB0", env="ESP32_SERIAL_PORT")
    esp32_baud_rate: int = Field(default=115200, env="ESP32_BAUD_RATE")
    mqtt_broker: str = Field(default="localhost", env="MQTT_BROKER")
    mqtt_port: int = Field(default=1883, env="MQTT_PORT")


class SearchSettings(BaseSettings):
    """Web search configuration."""
    
    engine: str = Field(default="duckduckgo", env="SEARCH_ENGINE")
    max_results: int = Field(default=5, env="MAX_SEARCH_RESULTS")
    timeout: int = Field(default=30, env="SEARCH_TIMEOUT")


class MemorySettings(BaseSettings):
    """Memory and RAG configuration."""
    
    vector_db: str = Field(default="chroma", env="MEMORY_VECTOR_DB")
    chunk_size: int = Field(default=1000, env="MEMORY_CHUNK_SIZE")
    chunk_overlap: int = Field(default=200, env="MEMORY_CHUNK_OVERLAP")
    embedding_model: str = Field(default="sentence-transformers/all-MiniLM-L6-v2", env="EMBEDDING_MODEL")


class LoggingSettings(BaseSettings):
    """Logging configuration."""
    
    level: str = Field(default="INFO", env="LOG_LEVEL")
    format: str = Field(default="json", env="LOG_FORMAT")
    file: str = Field(default="logs/xarvis.log", env="LOG_FILE")


class SecuritySettings(BaseSettings):
    """Security configuration."""
    
    secret_key: str = Field(default="your_secret_key_here", env="SECRET_KEY")
    access_token_expire_minutes: int = Field(default=30, env="ACCESS_TOKEN_EXPIRE_MINUTES")


class Settings(BaseSettings):
    """Main settings class combining all configurations."""
    
    environment: str = Field(default="development", env="ENVIRONMENT")
    debug: bool = Field(default=True, env="DEBUG")
    
    # Sub-configurations
    api: APISettings = Field(default_factory=APISettings)
    ai: AISettings = Field(default_factory=AISettings)
    database: DatabaseSettings = Field(default_factory=DatabaseSettings)
    redis: RedisSettings = Field(default_factory=RedisSettings)
    audio: AudioSettings = Field(default_factory=AudioSettings)
    hardware: HardwareSettings = Field(default_factory=HardwareSettings)
    search: SearchSettings = Field(default_factory=SearchSettings)
    memory: MemorySettings = Field(default_factory=MemorySettings)
    logging: LoggingSettings = Field(default_factory=LoggingSettings)
    security: SecuritySettings = Field(default_factory=SecuritySettings)
    
    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"
        case_sensitive = False


# Global settings instance
settings = Settings()


def get_settings() -> Settings:
    """Get the global settings instance."""
    return settings
