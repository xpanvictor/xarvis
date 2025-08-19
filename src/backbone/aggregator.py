"""
Aggregator - Data collection and preprocessing system.
Gathers information from various sources and prepares data for processing.
"""

from typing import Dict, Any, List, Optional, Union
from datetime import datetime, timedelta
from dataclasses import dataclass
from enum import Enum
import asyncio
import json
from pathlib import Path

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


class DataSource(Enum):
    """Types of data sources."""
    USER_INPUT = "user_input"
    AUDIO_INPUT = "audio_input"
    MEMORY_CONTEXT = "memory_context"
    WEB_SEARCH = "web_search"
    SYSTEM_STATUS = "system_status"
    HARDWARE_SENSORS = "hardware_sensors"
    SCHEDULED_TASK = "scheduled_task"


@dataclass
class AggregatedData:
    """Represents aggregated and preprocessed data."""
    primary_input: str
    context: Dict[str, Any]
    sources: List[DataSource]
    metadata: Dict[str, Any]
    timestamp: datetime
    confidence: float


class Aggregator:
    """
    Data aggregation and preprocessing system.
    Collects information from various sources and prepares it for the Brain.
    """
    
    def __init__(self, memory_system=None):
        self.memory_system = memory_system
        self.data_cache: Dict[str, Any] = {}
        self.source_handlers = {
            DataSource.USER_INPUT: self._process_user_input,
            DataSource.AUDIO_INPUT: self._process_audio_input,
            DataSource.MEMORY_CONTEXT: self._process_memory_context,
            DataSource.WEB_SEARCH: self._process_web_search,
            DataSource.SYSTEM_STATUS: self._process_system_status,
            DataSource.HARDWARE_SENSORS: self._process_hardware_sensors,
            DataSource.SCHEDULED_TASK: self._process_scheduled_task,
        }
        
    async def process(
        self,
        input_data: Dict[str, Any],
        metadata: Dict[str, Any]
    ) -> AggregatedData:
        """
        Process and aggregate input data from multiple sources.
        
        Args:
            input_data: Raw input data
            metadata: Request metadata
        
        Returns:
            Aggregated and preprocessed data
        """
        correlation_id = metadata.get("correlation_id", "unknown")
        
        with with_correlation_id(correlation_id):
            logger.info("Starting data aggregation", 
                       input_types=list(input_data.keys()))
            
            try:
                # Determine primary input and data sources
                primary_input, sources = await self._identify_sources(input_data)
                
                # Gather contextual data
                context = await self._gather_context(primary_input, sources, metadata)
                
                # Process each data source
                processed_data = {}
                for source in sources:
                    handler = self.source_handlers.get(source)
                    if handler:
                        processed_data[source.value] = await handler(
                            input_data, context, metadata
                        )
                
                # Calculate confidence score
                confidence = await self._calculate_confidence(processed_data, sources)
                
                # Create aggregated data object
                aggregated = AggregatedData(
                    primary_input=primary_input,
                    context=context,
                    sources=sources,
                    metadata={
                        **metadata,
                        "processed_data": processed_data,
                        "processing_timestamp": datetime.now().isoformat()
                    },
                    timestamp=datetime.now(),
                    confidence=confidence
                )
                
                logger.info("Data aggregation completed", 
                           sources_count=len(sources),
                           confidence=confidence)
                
                return aggregated
                
            except Exception as e:
                logger.error("Data aggregation failed", error=str(e))
                raise
    
    async def _identify_sources(
        self,
        input_data: Dict[str, Any]
    ) -> tuple[str, List[DataSource]]:
        """
        Identify the primary input and relevant data sources.
        
        Args:
            input_data: Raw input data
        
        Returns:
            Tuple of (primary_input, list_of_sources)
        """
        sources = []
        primary_input = ""
        
        # Check for text input
        if input_data.get("text"):
            primary_input = input_data["text"]
            sources.append(DataSource.USER_INPUT)
        
        # Check for audio input
        if input_data.get("audio"):
            sources.append(DataSource.AUDIO_INPUT)
            if not primary_input:
                primary_input = "[Audio Input]"
        
        # Always include memory context for conversation continuity
        sources.append(DataSource.MEMORY_CONTEXT)
        
        # Check if we need web search (based on input analysis)
        if await self._needs_web_search(primary_input):
            sources.append(DataSource.WEB_SEARCH)
        
        # Include system status for system-related queries
        if await self._is_system_query(primary_input):
            sources.append(DataSource.SYSTEM_STATUS)
        
        return primary_input, sources
    
    async def _gather_context(
        self,
        primary_input: str,
        sources: List[DataSource],
        metadata: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Gather contextual information for processing.
        
        Args:
            primary_input: Primary input text
            sources: List of data sources
            metadata: Request metadata
        
        Returns:
            Context dictionary
        """
        context = {
            "timestamp": datetime.now().isoformat(),
            "session_id": metadata.get("session_id"),
            "user_id": metadata.get("user_id"),
            "conversation_history": [],
            "system_context": {},
            "environmental_context": {}
        }
        
        # Gather conversation history from memory
        if DataSource.MEMORY_CONTEXT in sources and self.memory_system:
            try:
                history = await self.memory_system.get_conversation_history(
                    session_id=context["session_id"],
                    limit=10
                )
                context["conversation_history"] = history
            except Exception as e:
                logger.error("Failed to get conversation history", error=str(e))
        
        # Gather system context
        if DataSource.SYSTEM_STATUS in sources:
            context["system_context"] = await self._get_system_context()
        
        # Gather environmental context (time, date, etc.)
        context["environmental_context"] = await self._get_environmental_context()
        
        return context
    
    async def _process_user_input(
        self,
        input_data: Dict[str, Any],
        context: Dict[str, Any],
        metadata: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Process user text input."""
        text = input_data.get("text", "")
        
        return {
            "text": text,
            "length": len(text),
            "words": len(text.split()) if text else 0,
            "language": await self._detect_language(text),
            "sentiment": await self._analyze_sentiment(text),
            "intent": await self._extract_intent(text),
            "entities": await self._extract_entities(text)
        }
    
    async def _process_audio_input(
        self,
        input_data: Dict[str, Any],
        context: Dict[str, Any],
        metadata: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Process audio input data."""
        audio_data = input_data.get("audio")
        
        if not audio_data:
            return {"error": "No audio data provided"}
        
        return {
            "has_audio": True,
            "audio_length": len(audio_data) if isinstance(audio_data, bytes) else 0,
            "transcribed": False,  # Will be handled by audio interface
            "quality_check": await self._check_audio_quality(audio_data)
        }
    
    async def _process_memory_context(
        self,
        input_data: Dict[str, Any],
        context: Dict[str, Any],
        metadata: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Process memory and context information."""
        if not self.memory_system:
            return {"error": "Memory system not available"}
        
        try:
            primary_input = input_data.get("text", "")
            
            # Search for relevant memories
            relevant_memories = await self.memory_system.search(
                query=primary_input,
                limit=5
            ) if primary_input else []
            
            # Get recent context
            recent_context = await self.memory_system.get_recent_context(
                session_id=context.get("session_id"),
                limit=3
            )
            
            return {
                "relevant_memories": relevant_memories,
                "recent_context": recent_context,
                "memory_count": len(relevant_memories),
                "context_availability": len(recent_context) > 0
            }
            
        except Exception as e:
            logger.error("Failed to process memory context", error=str(e))
            return {"error": str(e)}
    
    async def _process_web_search(
        self,
        input_data: Dict[str, Any],
        context: Dict[str, Any],
        metadata: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Process web search requirements."""
        primary_input = input_data.get("text", "")
        
        # Extract search queries from input
        search_queries = await self._extract_search_queries(primary_input)
        
        return {
            "needs_search": len(search_queries) > 0,
            "search_queries": search_queries,
            "search_priority": await self._calculate_search_priority(primary_input)
        }
    
    async def _process_system_status(
        self,
        input_data: Dict[str, Any],
        context: Dict[str, Any],
        metadata: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Process system status information."""
        return await self._get_system_context()
    
    async def _process_hardware_sensors(
        self,
        input_data: Dict[str, Any],
        context: Dict[str, Any],
        metadata: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Process hardware sensor data."""
        # Placeholder for hardware sensor integration
        return {
            "sensors_available": False,
            "last_reading": None,
            "sensor_status": "not_configured"
        }
    
    async def _process_scheduled_task(
        self,
        input_data: Dict[str, Any],
        context: Dict[str, Any],
        metadata: Dict[str, Any]
    ) -> Dict[str, Any]:
        """Process scheduled task data."""
        return {
            "is_scheduled": metadata.get("scheduled", False),
            "scheduled_at": metadata.get("scheduled_at"),
            "task_type": metadata.get("task_type", "unknown")
        }
    
    # Helper methods
    async def _needs_web_search(self, input_text: str) -> bool:
        """Determine if input requires web search."""
        if not input_text:
            return False
        
        search_indicators = [
            "search", "look up", "find", "what is", "who is", 
            "when did", "where is", "how to", "latest", "current",
            "news", "weather", "stock", "price"
        ]
        
        input_lower = input_text.lower()
        return any(indicator in input_lower for indicator in search_indicators)
    
    async def _is_system_query(self, input_text: str) -> bool:
        """Determine if input is a system-related query."""
        if not input_text:
            return False
        
        system_indicators = [
            "status", "health", "system", "performance", "memory",
            "cpu", "disk", "network", "restart", "shutdown", "update"
        ]
        
        input_lower = input_text.lower()
        return any(indicator in input_lower for indicator in system_indicators)
    
    async def _get_system_context(self) -> Dict[str, Any]:
        """Get current system context."""
        return {
            "timestamp": datetime.now().isoformat(),
            "uptime": "unknown",  # Would be calculated
            "memory_usage": "unknown",
            "cpu_usage": "unknown",
            "active_tasks": 0,
            "system_health": "healthy"
        }
    
    async def _get_environmental_context(self) -> Dict[str, Any]:
        """Get environmental context (time, date, etc.)."""
        now = datetime.now()
        
        return {
            "current_time": now.isoformat(),
            "day_of_week": now.strftime("%A"),
            "date": now.strftime("%Y-%m-%d"),
            "time_of_day": self._get_time_period(now.hour),
            "timezone": str(now.astimezone().tzinfo)
        }
    
    def _get_time_period(self, hour: int) -> str:
        """Get time period description."""
        if 5 <= hour < 12:
            return "morning"
        elif 12 <= hour < 17:
            return "afternoon"
        elif 17 <= hour < 21:
            return "evening"
        else:
            return "night"
    
    async def _detect_language(self, text: str) -> str:
        """Detect language of input text."""
        # Placeholder - would use language detection library
        return "en" if text else "unknown"
    
    async def _analyze_sentiment(self, text: str) -> str:
        """Analyze sentiment of input text."""
        # Placeholder - would use sentiment analysis
        return "neutral" if text else "unknown"
    
    async def _extract_intent(self, text: str) -> str:
        """Extract intent from input text."""
        # Placeholder - would use NLU
        return "general_query" if text else "unknown"
    
    async def _extract_entities(self, text: str) -> List[Dict[str, Any]]:
        """Extract named entities from text."""
        # Placeholder - would use NER
        return []
    
    async def _check_audio_quality(self, audio_data: bytes) -> Dict[str, Any]:
        """Check audio quality metrics."""
        return {
            "quality": "good",
            "noise_level": "low",
            "duration": 0.0,
            "sample_rate": settings.audio.sample_rate
        }
    
    async def _extract_search_queries(self, text: str) -> List[str]:
        """Extract search queries from input text."""
        # Simple extraction - would be more sophisticated
        if await self._needs_web_search(text):
            return [text]
        return []
    
    async def _calculate_search_priority(self, text: str) -> int:
        """Calculate search priority (1-10)."""
        return 5  # Default medium priority
    
    async def _calculate_confidence(
        self,
        processed_data: Dict[str, Any],
        sources: List[DataSource]
    ) -> float:
        """Calculate confidence score for aggregated data."""
        base_confidence = 0.7
        
        # Adjust based on available data sources
        if len(sources) > 2:
            base_confidence += 0.1
        
        # Adjust based on data quality
        if DataSource.MEMORY_CONTEXT in sources:
            memory_data = processed_data.get(DataSource.MEMORY_CONTEXT.value, {})
            if memory_data.get("memory_count", 0) > 0:
                base_confidence += 0.1
        
        return min(1.0, base_confidence)
