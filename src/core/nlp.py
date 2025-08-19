"""
NLP Processor - Natural Language Processing components.
Handles text analysis, intent classification, and entity extraction.
"""

from typing import Dict, Any, List, Optional, Union, Tuple
from datetime import datetime
from dataclasses import dataclass
from enum import Enum
import re
import asyncio
from pathlib import Path

from src.utils.logging import get_logger
from src.config import settings


logger = get_logger(__name__)


class IntentType(Enum):
    """Common intent types."""
    QUESTION = "question"
    REQUEST = "request"
    COMMAND = "command"
    GREETING = "greeting"
    FAREWELL = "farewell"
    SEARCH = "search"
    CREATION = "creation"
    INFORMATION = "information"
    ACTION = "action"
    SYSTEM = "system"
    UNKNOWN = "unknown"


class EntityType(Enum):
    """Named entity types."""
    PERSON = "person"
    ORGANIZATION = "organization"
    LOCATION = "location"
    DATE = "date"
    TIME = "time"
    NUMBER = "number"
    EMAIL = "email"
    URL = "url"
    PHONE = "phone"
    CUSTOM = "custom"


@dataclass
class Entity:
    """Represents an extracted entity."""
    text: str
    entity_type: EntityType
    confidence: float
    start_pos: int
    end_pos: int
    normalized_value: Optional[str] = None


@dataclass
class NLPResult:
    """Results from NLP processing."""
    text: str
    intent: IntentType
    intent_confidence: float
    entities: List[Entity]
    sentiment: str
    sentiment_score: float
    language: str
    keywords: List[str]
    topics: List[str]
    complexity_score: float
    metadata: Dict[str, Any]


class NLPProcessor:
    """
    Natural Language Processing system for Xarvis.
    Provides text analysis, intent classification, and entity extraction.
    """
    
    def __init__(self):
        self.intent_patterns = self._load_intent_patterns()
        self.entity_patterns = self._load_entity_patterns()
        self.sentiment_keywords = self._load_sentiment_keywords()
        self.processing_cache: Dict[str, NLPResult] = {}
        
    async def process(
        self,
        text: str,
        context: Optional[Dict[str, Any]] = None
    ) -> NLPResult:
        """
        Process text through NLP pipeline.
        
        Args:
            text: Input text to process
            context: Optional context information
        
        Returns:
            NLP processing results
        """
        if not text or not text.strip():
            return self._create_empty_result(text)
        
        text = text.strip()
        
        # Check cache first
        cache_key = self._get_cache_key(text, context)
        if cache_key in self.processing_cache:
            logger.debug("Using cached NLP result", text_preview=text[:50])
            return self.processing_cache[cache_key]
        
        logger.debug("Processing text through NLP", 
                    text_length=len(text),
                    text_preview=text[:50] + "..." if len(text) > 50 else text)
        
        try:
            # Run NLP components in parallel where possible
            results = await asyncio.gather(
                self._classify_intent(text, context),
                self._extract_entities(text),
                self._analyze_sentiment(text),
                self._detect_language(text),
                self._extract_keywords(text),
                self._extract_topics(text),
                self._calculate_complexity(text),
                return_exceptions=True
            )
            
            # Handle any exceptions
            intent, intent_confidence = results[0] if not isinstance(results[0], Exception) else (IntentType.UNKNOWN, 0.0)
            entities = results[1] if not isinstance(results[1], Exception) else []
            sentiment, sentiment_score = results[2] if not isinstance(results[2], Exception) else ("neutral", 0.0)
            language = results[3] if not isinstance(results[3], Exception) else "en"
            keywords = results[4] if not isinstance(results[4], Exception) else []
            topics = results[5] if not isinstance(results[5], Exception) else []
            complexity = results[6] if not isinstance(results[6], Exception) else 0.5
            
            # Create result
            nlp_result = NLPResult(
                text=text,
                intent=intent,
                intent_confidence=intent_confidence,
                entities=entities,
                sentiment=sentiment,
                sentiment_score=sentiment_score,
                language=language,
                keywords=keywords,
                topics=topics,
                complexity_score=complexity,
                metadata={
                    "processed_at": datetime.now().isoformat(),
                    "processor_version": "1.0.0",
                    "context": context or {}
                }
            )
            
            # Cache result
            self.processing_cache[cache_key] = nlp_result
            
            logger.debug("NLP processing completed",
                        intent=intent.value,
                        intent_confidence=intent_confidence,
                        entities_count=len(entities),
                        sentiment=sentiment,
                        language=language)
            
            return nlp_result
            
        except Exception as e:
            logger.error("NLP processing failed", error=str(e))
            return self._create_error_result(text, str(e))
    
    async def _classify_intent(
        self,
        text: str,
        context: Optional[Dict[str, Any]] = None
    ) -> Tuple[IntentType, float]:
        """
        Classify the intent of the input text.
        
        Args:
            text: Input text
            context: Optional context
        
        Returns:
            Tuple of (intent, confidence_score)
        """
        text_lower = text.lower().strip()
        
        # Check for greeting patterns
        if any(pattern in text_lower for pattern in self.intent_patterns["greeting"]):
            return IntentType.GREETING, 0.9
        
        # Check for farewell patterns
        if any(pattern in text_lower for pattern in self.intent_patterns["farewell"]):
            return IntentType.FAREWELL, 0.9
        
        # Check for question patterns
        if (text.endswith('?') or 
            any(pattern in text_lower for pattern in self.intent_patterns["question"])):
            return IntentType.QUESTION, 0.8
        
        # Check for search patterns
        if any(pattern in text_lower for pattern in self.intent_patterns["search"]):
            return IntentType.SEARCH, 0.8
        
        # Check for creation patterns
        if any(pattern in text_lower for pattern in self.intent_patterns["creation"]):
            return IntentType.CREATION, 0.8
        
        # Check for action patterns
        if any(pattern in text_lower for pattern in self.intent_patterns["action"]):
            return IntentType.ACTION, 0.8
        
        # Check for system patterns
        if any(pattern in text_lower for pattern in self.intent_patterns["system"]):
            return IntentType.SYSTEM, 0.8
        
        # Check for command patterns (imperative mood)
        if self._is_command(text):
            return IntentType.COMMAND, 0.7
        
        # Default to request if none match
        return IntentType.REQUEST, 0.5
    
    async def _extract_entities(self, text: str) -> List[Entity]:
        """
        Extract named entities from text.
        
        Args:
            text: Input text
        
        Returns:
            List of extracted entities
        """
        entities = []
        
        # Extract different entity types using regex patterns
        for entity_type, patterns in self.entity_patterns.items():
            for pattern in patterns:
                matches = re.finditer(pattern, text, re.IGNORECASE)
                for match in matches:
                    entity = Entity(
                        text=match.group(),
                        entity_type=EntityType(entity_type),
                        confidence=0.8,  # Rule-based confidence
                        start_pos=match.start(),
                        end_pos=match.end(),
                        normalized_value=self._normalize_entity(match.group(), entity_type)
                    )
                    entities.append(entity)
        
        # Remove overlapping entities (keep higher confidence ones)
        entities = self._remove_overlapping_entities(entities)
        
        return entities
    
    async def _analyze_sentiment(self, text: str) -> Tuple[str, float]:
        """
        Analyze sentiment of the text.
        
        Args:
            text: Input text
        
        Returns:
            Tuple of (sentiment_label, confidence_score)
        """
        text_lower = text.lower()
        
        positive_score = sum(1 for word in self.sentiment_keywords["positive"] if word in text_lower)
        negative_score = sum(1 for word in self.sentiment_keywords["negative"] if word in text_lower)
        
        total_sentiment_words = positive_score + negative_score
        
        if total_sentiment_words == 0:
            return "neutral", 0.5
        
        if positive_score > negative_score:
            confidence = positive_score / len(text.split()) * 10  # Normalize
            return "positive", min(0.9, max(0.6, confidence))
        elif negative_score > positive_score:
            confidence = negative_score / len(text.split()) * 10  # Normalize
            return "negative", min(0.9, max(0.6, confidence))
        else:
            return "neutral", 0.7
    
    async def _detect_language(self, text: str) -> str:
        """
        Detect the language of the text.
        
        Args:
            text: Input text
        
        Returns:
            Language code
        """
        # Simple language detection based on common words
        english_indicators = ["the", "and", "is", "are", "was", "were", "have", "has", "do", "does"]
        spanish_indicators = ["el", "la", "es", "son", "fue", "fueron", "tiene", "hacer"]
        french_indicators = ["le", "la", "est", "sont", "était", "étaient", "avoir", "faire"]
        
        text_lower = text.lower()
        words = text_lower.split()
        
        english_score = sum(1 for word in words if word in english_indicators)
        spanish_score = sum(1 for word in words if word in spanish_indicators)
        french_score = sum(1 for word in words if word in french_indicators)
        
        if english_score >= spanish_score and english_score >= french_score:
            return "en"
        elif spanish_score >= french_score:
            return "es"
        elif french_score > 0:
            return "fr"
        else:
            return "en"  # Default to English
    
    async def _extract_keywords(self, text: str) -> List[str]:
        """
        Extract keywords from text.
        
        Args:
            text: Input text
        
        Returns:
            List of keywords
        """
        # Simple keyword extraction based on word frequency and length
        words = re.findall(r'\b\w+\b', text.lower())
        
        # Filter out common stop words
        stop_words = {
            "the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with",
            "by", "a", "an", "is", "are", "was", "were", "be", "been", "have", "has",
            "had", "do", "does", "did", "will", "would", "could", "should", "can",
            "may", "might", "must", "i", "you", "he", "she", "it", "we", "they",
            "me", "him", "her", "us", "them", "my", "your", "his", "her", "its",
            "our", "their", "this", "that", "these", "those"
        }
        
        # Get meaningful words
        keywords = []
        for word in words:
            if (len(word) > 3 and 
                word not in stop_words and 
                not word.isdigit()):
                keywords.append(word)
        
        # Return unique keywords, sorted by frequency
        from collections import Counter
        word_counts = Counter(keywords)
        return [word for word, count in word_counts.most_common(10)]
    
    async def _extract_topics(self, text: str) -> List[str]:
        """
        Extract topics from text.
        
        Args:
            text: Input text
        
        Returns:
            List of topics
        """
        # Simple topic extraction based on keyword patterns
        topics = []
        text_lower = text.lower()
        
        topic_keywords = {
            "technology": ["computer", "software", "programming", "ai", "machine", "learning", "code", "system"],
            "business": ["company", "business", "market", "sales", "revenue", "profit", "customer"],
            "science": ["research", "study", "experiment", "data", "analysis", "theory", "hypothesis"],
            "health": ["health", "medical", "doctor", "patient", "treatment", "medicine", "disease"],
            "entertainment": ["movie", "music", "game", "book", "show", "entertainment", "fun"],
            "sports": ["sport", "game", "team", "player", "match", "competition", "athletic"],
            "travel": ["travel", "trip", "vacation", "hotel", "flight", "destination", "tourism"],
            "food": ["food", "restaurant", "recipe", "cook", "meal", "eat", "dish", "cuisine"]
        }
        
        for topic, keywords in topic_keywords.items():
            if any(keyword in text_lower for keyword in keywords):
                topics.append(topic)
        
        return topics
    
    async def _calculate_complexity(self, text: str) -> float:
        """
        Calculate text complexity score.
        
        Args:
            text: Input text
        
        Returns:
            Complexity score (0-1)
        """
        if not text:
            return 0.0
        
        # Factors contributing to complexity
        word_count = len(text.split())
        sentence_count = len([s for s in text.split('.') if s.strip()])
        avg_word_length = sum(len(word) for word in text.split()) / word_count if word_count > 0 else 0
        
        # Calculate complexity score
        complexity = 0.0
        
        # Word count factor
        if word_count > 50:
            complexity += 0.3
        elif word_count > 20:
            complexity += 0.2
        elif word_count > 10:
            complexity += 0.1
        
        # Average word length factor
        if avg_word_length > 7:
            complexity += 0.3
        elif avg_word_length > 5:
            complexity += 0.2
        
        # Sentence complexity factor
        if sentence_count > 0:
            avg_words_per_sentence = word_count / sentence_count
            if avg_words_per_sentence > 20:
                complexity += 0.4
            elif avg_words_per_sentence > 15:
                complexity += 0.2
        
        return min(1.0, complexity)
    
    def _is_command(self, text: str) -> bool:
        """Check if text is likely a command."""
        text = text.strip().lower()
        
        # Commands often start with verbs
        command_starters = [
            "run", "execute", "start", "stop", "create", "delete", "update",
            "show", "display", "open", "close", "save", "load", "install",
            "remove", "add", "set", "get", "find", "search", "help", "tell",
            "please", "can you", "could you", "would you"
        ]
        
        return any(text.startswith(starter) for starter in command_starters)
    
    def _normalize_entity(self, entity_text: str, entity_type: str) -> Optional[str]:
        """Normalize entity value based on type."""
        if entity_type == "email":
            return entity_text.lower()
        elif entity_type == "url":
            return entity_text.lower()
        elif entity_type == "phone":
            # Remove non-numeric characters
            return re.sub(r'[^\d]', '', entity_text)
        elif entity_type == "date":
            # This would normally parse and normalize dates
            return entity_text
        else:
            return entity_text
    
    def _remove_overlapping_entities(self, entities: List[Entity]) -> List[Entity]:
        """Remove overlapping entities, keeping higher confidence ones."""
        if not entities:
            return []
        
        # Sort by start position
        entities.sort(key=lambda e: e.start_pos)
        
        filtered = []
        for entity in entities:
            # Check if it overlaps with any already added entity
            overlaps = False
            for existing in filtered:
                if (entity.start_pos < existing.end_pos and 
                    entity.end_pos > existing.start_pos):
                    overlaps = True
                    # Keep the higher confidence entity
                    if entity.confidence > existing.confidence:
                        filtered.remove(existing)
                        filtered.append(entity)
                    break
            
            if not overlaps:
                filtered.append(entity)
        
        return filtered
    
    def _get_cache_key(self, text: str, context: Optional[Dict[str, Any]]) -> str:
        """Generate cache key for text and context."""
        import hashlib
        
        context_str = str(sorted(context.items())) if context else ""
        combined = f"{text}|{context_str}"
        
        return hashlib.md5(combined.encode()).hexdigest()
    
    def _create_empty_result(self, text: str) -> NLPResult:
        """Create empty NLP result for invalid input."""
        return NLPResult(
            text=text,
            intent=IntentType.UNKNOWN,
            intent_confidence=0.0,
            entities=[],
            sentiment="neutral",
            sentiment_score=0.0,
            language="en",
            keywords=[],
            topics=[],
            complexity_score=0.0,
            metadata={"error": "Empty or invalid input"}
        )
    
    def _create_error_result(self, text: str, error: str) -> NLPResult:
        """Create error NLP result."""
        return NLPResult(
            text=text,
            intent=IntentType.UNKNOWN,
            intent_confidence=0.0,
            entities=[],
            sentiment="neutral",
            sentiment_score=0.0,
            language="en",
            keywords=[],
            topics=[],
            complexity_score=0.0,
            metadata={"error": error}
        )
    
    def _load_intent_patterns(self) -> Dict[str, List[str]]:
        """Load intent classification patterns."""
        return {
            "greeting": ["hello", "hi", "hey", "good morning", "good afternoon", "good evening"],
            "farewell": ["goodbye", "bye", "see you", "farewell", "good night"],
            "question": ["what", "how", "why", "when", "where", "who", "which", "can you tell me"],
            "search": ["search", "find", "look up", "google", "search for"],
            "creation": ["create", "make", "generate", "build", "write", "compose"],
            "action": ["run", "execute", "start", "stop", "play", "pause", "do"],
            "system": ["status", "health", "system", "memory", "cpu", "performance"]
        }
    
    def _load_entity_patterns(self) -> Dict[str, List[str]]:
        """Load entity extraction patterns."""
        return {
            "email": [r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b'],
            "url": [r'https?://[^\s]+', r'www\.[^\s]+'],
            "phone": [r'\b\d{3}-\d{3}-\d{4}\b', r'\(\d{3}\)\s*\d{3}-\d{4}'],
            "date": [r'\b\d{1,2}/\d{1,2}/\d{4}\b', r'\b\d{4}-\d{2}-\d{2}\b'],
            "time": [r'\b\d{1,2}:\d{2}(?::\d{2})?\s*(?:AM|PM|am|pm)?\b'],
            "number": [r'\b\d+(?:\.\d+)?\b']
        }
    
    def _load_sentiment_keywords(self) -> Dict[str, List[str]]:
        """Load sentiment analysis keywords."""
        return {
            "positive": [
                "good", "great", "excellent", "amazing", "wonderful", "fantastic",
                "perfect", "awesome", "brilliant", "outstanding", "superb",
                "happy", "pleased", "satisfied", "delighted", "thrilled",
                "love", "like", "enjoy", "appreciate", "thank"
            ],
            "negative": [
                "bad", "terrible", "awful", "horrible", "disgusting", "hate",
                "dislike", "disappointed", "frustrated", "angry", "annoyed",
                "sad", "unhappy", "upset", "worried", "concerned",
                "wrong", "error", "problem", "issue", "trouble", "fail"
            ]
        }
    
    def clear_cache(self) -> None:
        """Clear the processing cache."""
        self.processing_cache.clear()
        logger.info("NLP processing cache cleared")
    
    def get_stats(self) -> Dict[str, Any]:
        """Get NLP processor statistics."""
        return {
            "cache_size": len(self.processing_cache),
            "intent_patterns": len(self.intent_patterns),
            "entity_patterns": sum(len(patterns) for patterns in self.entity_patterns.values()),
            "sentiment_keywords": sum(len(keywords) for keywords in self.sentiment_keywords.values())
        }
