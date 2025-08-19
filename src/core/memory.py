"""
Memory System - RAG-based memory and context management.
Handles conversation history, knowledge storage, and contextual retrieval.
"""

from typing import Dict, Any, List, Optional, Union
from datetime import datetime, timedelta
from dataclasses import dataclass
from enum import Enum
import json
import uuid
import asyncio
from pathlib import Path

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


class MemoryType(Enum):
    """Types of memories stored in the system."""
    CONVERSATION = "conversation"
    FACTUAL = "factual"
    PROCEDURAL = "procedural"
    EPISODIC = "episodic"
    SYSTEM = "system"


@dataclass
class Memory:
    """Represents a memory entry."""
    id: str
    type: MemoryType
    content: str
    metadata: Dict[str, Any]
    timestamp: datetime
    importance: float
    access_count: int
    last_accessed: datetime
    embedding: Optional[List[float]] = None


@dataclass
class SearchResult:
    """Represents a memory search result."""
    memory: Memory
    relevance_score: float
    context_match: float


class MemorySystem:
    """
    RAG-based memory system for storing and retrieving contextual information.
    Maintains conversation history and provides intelligent memory recall.
    """
    
    def __init__(self):
        self.vector_db = None
        self.embedding_model = None
        self.conversation_sessions: Dict[str, List[Memory]] = {}
        self.memory_index: Dict[str, Memory] = {}
        self.data_path = Path("data/memory")
        
    async def initialize(self) -> None:
        """Initialize the memory system."""
        logger.info("Initializing Memory System")
        
        try:
            # Create data directory
            self.data_path.mkdir(parents=True, exist_ok=True)
            
            # Initialize embedding model
            await self._initialize_embedding_model()
            
            # Initialize vector database
            await self._initialize_vector_db()
            
            # Load existing memories
            await self._load_memories()
            
            logger.info("Memory System initialized successfully")
            
        except Exception as e:
            logger.error("Failed to initialize Memory System", error=str(e))
            raise
    
    async def store(
        self,
        content: str,
        memory_type: MemoryType = MemoryType.FACTUAL,
        metadata: Optional[Dict[str, Any]] = None,
        importance: float = 0.5,
        session_id: Optional[str] = None
    ) -> str:
        """
        Store information in memory.
        
        Args:
            content: Content to store
            memory_type: Type of memory
            metadata: Additional metadata
            importance: Importance score (0-1)
            session_id: Session ID for conversation memories
        
        Returns:
            Memory ID
        """
        memory_id = str(uuid.uuid4())
        
        try:
            # Generate embedding for the content
            embedding = await self._generate_embedding(content)
            
            # Create memory object
            memory = Memory(
                id=memory_id,
                type=memory_type,
                content=content,
                metadata=metadata or {},
                timestamp=datetime.now(),
                importance=importance,
                access_count=0,
                last_accessed=datetime.now(),
                embedding=embedding
            )
            
            # Store in memory index
            self.memory_index[memory_id] = memory
            
            # Store in vector database
            if self.vector_db:
                await self._store_in_vector_db(memory)
            
            # Add to session if provided
            if session_id and memory_type == MemoryType.CONVERSATION:
                if session_id not in self.conversation_sessions:
                    self.conversation_sessions[session_id] = []
                self.conversation_sessions[session_id].append(memory)
            
            # Persist to disk
            await self._persist_memory(memory)
            
            logger.debug("Memory stored", 
                        memory_id=memory_id, 
                        type=memory_type.value,
                        importance=importance)
            
            return memory_id
            
        except Exception as e:
            logger.error("Failed to store memory", error=str(e))
            raise
    
    async def search(
        self,
        query: str,
        limit: int = 10,
        memory_types: Optional[List[MemoryType]] = None,
        min_relevance: float = 0.5,
        session_id: Optional[str] = None
    ) -> List[SearchResult]:
        """
        Search memories using semantic similarity.
        
        Args:
            query: Search query
            limit: Maximum results to return
            memory_types: Filter by memory types
            min_relevance: Minimum relevance score
            session_id: Filter by session ID
        
        Returns:
            List of search results
        """
        try:
            logger.debug("Searching memories", 
                        query=query[:50] + "...", 
                        limit=limit)
            
            # Generate query embedding
            query_embedding = await self._generate_embedding(query)
            
            # Search in vector database if available
            if self.vector_db and query_embedding:
                return await self._search_vector_db(
                    query_embedding, limit, memory_types, min_relevance, session_id
                )
            
            # Fallback to simple text matching
            return await self._search_fallback(
                query, limit, memory_types, min_relevance, session_id
            )
            
        except Exception as e:
            logger.error("Memory search failed", error=str(e))
            return []
    
    async def get_conversation_history(
        self,
        session_id: str,
        limit: int = 50
    ) -> List[Dict[str, Any]]:
        """
        Get conversation history for a session.
        
        Args:
            session_id: Session ID
            limit: Maximum entries to return
        
        Returns:
            List of conversation entries
        """
        try:
            if session_id not in self.conversation_sessions:
                return []
            
            memories = self.conversation_sessions[session_id][-limit:]
            
            history = []
            for memory in memories:
                # Update access count
                memory.access_count += 1
                memory.last_accessed = datetime.now()
                
                history.append({
                    "id": memory.id,
                    "content": memory.content,
                    "timestamp": memory.timestamp.isoformat(),
                    "metadata": memory.metadata,
                    "importance": memory.importance
                })
            
            return history
            
        except Exception as e:
            logger.error("Failed to get conversation history", error=str(e))
            return []
    
    async def get_recent_context(
        self,
        session_id: Optional[str] = None,
        limit: int = 5,
        time_window: timedelta = timedelta(hours=1)
    ) -> List[Dict[str, Any]]:
        """
        Get recent contextual information.
        
        Args:
            session_id: Optional session ID to filter by
            limit: Maximum entries to return
            time_window: Time window to consider
        
        Returns:
            List of recent context entries
        """
        try:
            cutoff_time = datetime.now() - time_window
            recent_memories = []
            
            for memory in self.memory_index.values():
                if memory.timestamp >= cutoff_time:
                    if session_id is None or memory.metadata.get("session_id") == session_id:
                        recent_memories.append(memory)
            
            # Sort by timestamp and importance
            recent_memories.sort(
                key=lambda m: (m.timestamp, m.importance), 
                reverse=True
            )
            
            return [
                {
                    "id": memory.id,
                    "content": memory.content,
                    "type": memory.type.value,
                    "timestamp": memory.timestamp.isoformat(),
                    "importance": memory.importance,
                    "metadata": memory.metadata
                }
                for memory in recent_memories[:limit]
            ]
            
        except Exception as e:
            logger.error("Failed to get recent context", error=str(e))
            return []
    
    async def store_interaction(
        self,
        session_id: str,
        user_input: str,
        assistant_response: str,
        metadata: Optional[Dict[str, Any]] = None
    ) -> None:
        """
        Store a conversation interaction.
        
        Args:
            session_id: Session ID
            user_input: User's input
            assistant_response: Assistant's response
            metadata: Additional metadata
        """
        try:
            interaction_metadata = {
                "session_id": session_id,
                "interaction_type": "conversation",
                **(metadata or {})
            }
            
            # Store user input
            await self.store(
                content=f"User: {user_input}",
                memory_type=MemoryType.CONVERSATION,
                metadata={**interaction_metadata, "speaker": "user"},
                importance=0.6,
                session_id=session_id
            )
            
            # Store assistant response
            await self.store(
                content=f"Assistant: {assistant_response}",
                memory_type=MemoryType.CONVERSATION,
                metadata={**interaction_metadata, "speaker": "assistant"},
                importance=0.6,
                session_id=session_id
            )
            
        except Exception as e:
            logger.error("Failed to store interaction", error=str(e))
    
    async def update_memory_importance(
        self,
        memory_id: str,
        importance: float
    ) -> bool:
        """
        Update the importance of a memory.
        
        Args:
            memory_id: Memory ID
            importance: New importance score (0-1)
        
        Returns:
            True if successful
        """
        try:
            if memory_id in self.memory_index:
                self.memory_index[memory_id].importance = importance
                await self._persist_memory(self.memory_index[memory_id])
                return True
            return False
            
        except Exception as e:
            logger.error("Failed to update memory importance", error=str(e))
            return False
    
    async def delete_memory(self, memory_id: str) -> bool:
        """
        Delete a memory.
        
        Args:
            memory_id: Memory ID to delete
        
        Returns:
            True if successful
        """
        try:
            if memory_id in self.memory_index:
                memory = self.memory_index[memory_id]
                
                # Remove from vector database
                if self.vector_db:
                    await self._delete_from_vector_db(memory_id)
                
                # Remove from session
                for session_memories in self.conversation_sessions.values():
                    session_memories[:] = [m for m in session_memories if m.id != memory_id]
                
                # Remove from index
                del self.memory_index[memory_id]
                
                # Remove from disk
                await self._delete_persisted_memory(memory_id)
                
                return True
            
            return False
            
        except Exception as e:
            logger.error("Failed to delete memory", error=str(e))
            return False
    
    async def cleanup_old_memories(
        self,
        max_age: timedelta = timedelta(days=30),
        min_importance: float = 0.3
    ) -> int:
        """
        Clean up old, unimportant memories.
        
        Args:
            max_age: Maximum age for memories
            min_importance: Minimum importance to keep
        
        Returns:
            Number of memories cleaned up
        """
        cutoff_time = datetime.now() - max_age
        cleaned_count = 0
        
        memories_to_delete = []
        
        for memory in self.memory_index.values():
            if (memory.last_accessed < cutoff_time and 
                memory.importance < min_importance and
                memory.type != MemoryType.SYSTEM):
                memories_to_delete.append(memory.id)
        
        for memory_id in memories_to_delete:
            if await self.delete_memory(memory_id):
                cleaned_count += 1
        
        logger.info("Memory cleanup completed", cleaned_count=cleaned_count)
        return cleaned_count
    
    async def _initialize_embedding_model(self) -> None:
        """Initialize the embedding model."""
        try:
            from sentence_transformers import SentenceTransformer
            
            self.embedding_model = SentenceTransformer(
                settings.memory.embedding_model
            )
            logger.info("Embedding model initialized")
            
        except ImportError:
            logger.warning("sentence-transformers not available, using fallback")
            self.embedding_model = None
        except Exception as e:
            logger.error("Failed to initialize embedding model", error=str(e))
            self.embedding_model = None
    
    async def _initialize_vector_db(self) -> None:
        """Initialize the vector database."""
        try:
            if settings.memory.vector_db.lower() == "chroma":
                import chromadb
                
                self.vector_db = chromadb.PersistentClient(
                    path=str(self.data_path / "chroma")
                )
                logger.info("ChromaDB initialized")
            
            elif settings.memory.vector_db.lower() == "faiss":
                # FAISS initialization would go here
                logger.info("FAISS not implemented yet, using fallback")
                self.vector_db = None
            
        except ImportError as e:
            logger.warning(f"Vector database not available: {e}")
            self.vector_db = None
        except Exception as e:
            logger.error("Failed to initialize vector database", error=str(e))
            self.vector_db = None
    
    async def _generate_embedding(self, text: str) -> Optional[List[float]]:
        """Generate embedding for text."""
        if not self.embedding_model or not text.strip():
            return None
        
        try:
            # Run embedding generation in thread pool
            loop = asyncio.get_event_loop()
            embedding = await loop.run_in_executor(
                None, 
                lambda: self.embedding_model.encode(text).tolist()
            )
            return embedding
            
        except Exception as e:
            logger.error("Failed to generate embedding", error=str(e))
            return None
    
    async def _store_in_vector_db(self, memory: Memory) -> None:
        """Store memory in vector database."""
        if not self.vector_db or not memory.embedding:
            return
        
        try:
            collection = self.vector_db.get_or_create_collection(
                name=f"memories_{memory.type.value}"
            )
            
            collection.add(
                documents=[memory.content],
                embeddings=[memory.embedding],
                metadatas=[{
                    "memory_id": memory.id,
                    "timestamp": memory.timestamp.isoformat(),
                    "importance": memory.importance,
                    **memory.metadata
                }],
                ids=[memory.id]
            )
            
        except Exception as e:
            logger.error("Failed to store in vector DB", error=str(e))
    
    async def _search_vector_db(
        self,
        query_embedding: List[float],
        limit: int,
        memory_types: Optional[List[MemoryType]],
        min_relevance: float,
        session_id: Optional[str]
    ) -> List[SearchResult]:
        """Search vector database."""
        results = []
        
        try:
            types_to_search = memory_types or list(MemoryType)
            
            for memory_type in types_to_search:
                collection_name = f"memories_{memory_type.value}"
                
                try:
                    collection = self.vector_db.get_collection(collection_name)
                    
                    search_results = collection.query(
                        query_embeddings=[query_embedding],
                        n_results=limit,
                        where={"session_id": session_id} if session_id else None
                    )
                    
                    for i, doc_id in enumerate(search_results["ids"][0]):
                        relevance = 1 - search_results["distances"][0][i]
                        
                        if relevance >= min_relevance and doc_id in self.memory_index:
                            memory = self.memory_index[doc_id]
                            
                            results.append(SearchResult(
                                memory=memory,
                                relevance_score=relevance,
                                context_match=relevance  # Simplified
                            ))
                
                except Exception as e:
                    logger.debug(f"Collection {collection_name} not found or error: {e}")
            
            # Sort by relevance
            results.sort(key=lambda r: r.relevance_score, reverse=True)
            return results[:limit]
            
        except Exception as e:
            logger.error("Vector DB search failed", error=str(e))
            return []
    
    async def _search_fallback(
        self,
        query: str,
        limit: int,
        memory_types: Optional[List[MemoryType]],
        min_relevance: float,
        session_id: Optional[str]
    ) -> List[SearchResult]:
        """Fallback text-based search."""
        results = []
        query_lower = query.lower()
        
        for memory in self.memory_index.values():
            # Filter by type
            if memory_types and memory.type not in memory_types:
                continue
            
            # Filter by session
            if session_id and memory.metadata.get("session_id") != session_id:
                continue
            
            # Simple text matching
            content_lower = memory.content.lower()
            if query_lower in content_lower:
                # Calculate simple relevance score
                relevance = len(query_lower) / len(content_lower)
                relevance = min(1.0, relevance * 2)  # Boost score
                
                if relevance >= min_relevance:
                    results.append(SearchResult(
                        memory=memory,
                        relevance_score=relevance,
                        context_match=relevance
                    ))
        
        # Sort by relevance and importance
        results.sort(
            key=lambda r: (r.relevance_score, r.memory.importance), 
            reverse=True
        )
        
        return results[:limit]
    
    async def _load_memories(self) -> None:
        """Load existing memories from disk."""
        try:
            memory_file = self.data_path / "memories.jsonl"
            if not memory_file.exists():
                return
            
            with open(memory_file, 'r') as f:
                for line in f:
                    try:
                        data = json.loads(line.strip())
                        memory = self._deserialize_memory(data)
                        self.memory_index[memory.id] = memory
                        
                        # Add to session if conversation
                        if memory.type == MemoryType.CONVERSATION:
                            session_id = memory.metadata.get("session_id")
                            if session_id:
                                if session_id not in self.conversation_sessions:
                                    self.conversation_sessions[session_id] = []
                                self.conversation_sessions[session_id].append(memory)
                    
                    except Exception as e:
                        logger.error("Failed to load memory", error=str(e))
            
            logger.info("Memories loaded", count=len(self.memory_index))
            
        except Exception as e:
            logger.error("Failed to load memories", error=str(e))
    
    async def _persist_memory(self, memory: Memory) -> None:
        """Persist memory to disk."""
        try:
            memory_file = self.data_path / "memories.jsonl"
            
            with open(memory_file, 'a') as f:
                data = self._serialize_memory(memory)
                f.write(json.dumps(data) + '\n')
                
        except Exception as e:
            logger.error("Failed to persist memory", error=str(e))
    
    async def _delete_persisted_memory(self, memory_id: str) -> None:
        """Remove memory from persistent storage."""
        # This is simplified - in production, you'd want a more efficient approach
        try:
            memory_file = self.data_path / "memories.jsonl"
            if not memory_file.exists():
                return
            
            # Read all memories except the one to delete
            remaining_memories = []
            with open(memory_file, 'r') as f:
                for line in f:
                    data = json.loads(line.strip())
                    if data.get("id") != memory_id:
                        remaining_memories.append(line)
            
            # Rewrite file
            with open(memory_file, 'w') as f:
                f.writelines(remaining_memories)
                
        except Exception as e:
            logger.error("Failed to delete persisted memory", error=str(e))
    
    async def _delete_from_vector_db(self, memory_id: str) -> None:
        """Delete memory from vector database."""
        if not self.vector_db:
            return
        
        try:
            # Search all collections for the memory
            for memory_type in MemoryType:
                collection_name = f"memories_{memory_type.value}"
                try:
                    collection = self.vector_db.get_collection(collection_name)
                    collection.delete(ids=[memory_id])
                except Exception:
                    # Collection might not exist
                    pass
                    
        except Exception as e:
            logger.error("Failed to delete from vector DB", error=str(e))
    
    def _serialize_memory(self, memory: Memory) -> Dict[str, Any]:
        """Serialize memory to dictionary."""
        return {
            "id": memory.id,
            "type": memory.type.value,
            "content": memory.content,
            "metadata": memory.metadata,
            "timestamp": memory.timestamp.isoformat(),
            "importance": memory.importance,
            "access_count": memory.access_count,
            "last_accessed": memory.last_accessed.isoformat(),
            "embedding": memory.embedding
        }
    
    def _deserialize_memory(self, data: Dict[str, Any]) -> Memory:
        """Deserialize memory from dictionary."""
        return Memory(
            id=data["id"],
            type=MemoryType(data["type"]),
            content=data["content"],
            metadata=data["metadata"],
            timestamp=datetime.fromisoformat(data["timestamp"]),
            importance=data["importance"],
            access_count=data["access_count"],
            last_accessed=datetime.fromisoformat(data["last_accessed"]),
            embedding=data.get("embedding")
        )
    
    async def health_check(self) -> Dict[str, Any]:
        """Perform memory system health check."""
        return {
            "status": "healthy",
            "total_memories": len(self.memory_index),
            "active_sessions": len(self.conversation_sessions),
            "vector_db": self.vector_db is not None,
            "embedding_model": self.embedding_model is not None
        }
