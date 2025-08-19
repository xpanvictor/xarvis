"""
Brain Module - Central AI intelligence for reasoning and decision making.
Coordinates LLM interactions, tool usage, and intelligent response generation.
"""

import asyncio
from typing import Dict, Any, List, Optional, Tuple, AsyncGenerator
from datetime import datetime
import json
from enum import Enum

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings

from typing import Dict, Any, List, Optional, Union
from datetime import datetime
from dataclasses import dataclass
from enum import Enum
import asyncio
import json
import uuid

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


class ResponseType(Enum):
    """Types of responses the brain can generate."""
    TEXT_RESPONSE = "text_response"
    ACTION_SEQUENCE = "action_sequence"
    TOOL_CALL = "tool_call"
    CODE_GENERATION = "code_generation"
    SYSTEM_COMMAND = "system_command"
    AUDIO_RESPONSE = "audio_response"


class ProcessingMode(Enum):
    """Processing modes for different types of requests."""
    REALTIME = "realtime"
    BACKGROUND = "background"
    SCHEDULED = "scheduled"
    INTENSIVE = "intensive"


@dataclass
class BrainResponse:
    """Response from the brain processing."""
    response_id: str
    response_type: ResponseType
    content: str
    actions: List[Dict[str, Any]]
    tool_calls: List[Dict[str, Any]]
    context: Dict[str, Any]
    confidence: float
    processing_time: float
    metadata: Dict[str, Any]


class Brain:
    """
    The central AI processing unit for Xarvis.
    Responsible for understanding, reasoning, and generating responses.
    """
    
    def __init__(self, memory_system=None, nlp_processor=None, tool_registry=None):
        self.memory_system = memory_system
        self.nlp_processor = nlp_processor
        self.tool_registry = tool_registry
        self.conversation_context: Dict[str, Any] = {}
        self.active_sessions: Dict[str, Dict[str, Any]] = {}
        self.thinking_cache: Dict[str, Any] = {}
        
        # AI client initialization would happen here
        self.ai_client = None  # Will be initialized in initialize()
        
    async def initialize(self) -> None:
        """Initialize the brain system."""
        logger.info("Initializing Brain system")
        
        try:
            # Initialize AI client (OpenAI/Anthropic)
            await self._initialize_ai_client()
            
            # Load system prompts and instructions
            await self._load_system_prompts()
            
            # Initialize conversation tracking
            self.conversation_context = {}
            
            logger.info("Brain system initialized successfully")
            
        except Exception as e:
            logger.error("Failed to initialize Brain system", error=str(e))
            raise
    
    async def process(
        self,
        input_data: Any,  # AggregatedData from aggregator
        request_type: str,
        correlation_id: str,
        processing_mode: ProcessingMode = ProcessingMode.REALTIME
    ) -> Dict[str, Any]:
        """
        Process input through the brain's AI system.
        
        Args:
            input_data: Aggregated input data
            request_type: Type of request being processed
            correlation_id: Request correlation ID
            processing_mode: How to process the request
        
        Returns:
            Brain processing response
        """
        start_time = datetime.now()
        
        with with_correlation_id(correlation_id):
            logger.info("Starting brain processing", 
                       request_type=request_type,
                       processing_mode=processing_mode.value)
            
            try:
                # Extract primary input and context
                primary_input = getattr(input_data, 'primary_input', '')
                context = getattr(input_data, 'context', {})
                confidence = getattr(input_data, 'confidence', 0.7)
                
                # Get or create session
                session_id = context.get("session_id", "default")
                session_context = await self._get_session_context(session_id)
                
                # Analyze input and determine response strategy
                analysis = await self._analyze_input(primary_input, context)
                
                # Generate response based on analysis
                brain_response = await self._generate_response(
                    primary_input,
                    context,
                    analysis,
                    session_context,
                    processing_mode
                )
                
                # Update session context
                await self._update_session_context(session_id, {
                    "last_input": primary_input,
                    "last_response": brain_response.content,
                    "timestamp": datetime.now().isoformat()
                })
                
                # Store in memory only if analysis indicates it's valuable
                if (self.memory_system and 
                    analysis.get("should_store_in_memory", False) and
                    analysis.get("memory_importance", "low") in ["medium", "high"]):
                    await self._store_interaction(
                        session_id,
                        primary_input,
                        brain_response,
                        correlation_id,
                        analysis.get("memory_importance", "low")
                    )
                
                # Calculate processing time
                processing_time = (datetime.now() - start_time).total_seconds()
                
                # Format response
                formatted_response = {
                    "response": brain_response.content,
                    "response_type": brain_response.response_type.value,
                    "actions": brain_response.actions,
                    "tool_calls": brain_response.tool_calls,
                    "context": brain_response.context,
                    "confidence": brain_response.confidence,
                    "processing_time": processing_time,
                    "conversation_id": session_id,
                    "metadata": {
                        **brain_response.metadata,
                        "analysis": analysis,
                        "session_updated": True
                    }
                }
                
                # Check if response needs audio generation
                if self._needs_audio_response(analysis, brain_response):
                    formatted_response["audio_response"] = {
                        "text": brain_response.content,
                        "voice_settings": await self._get_voice_settings(context),
                        "priority": "normal"
                    }
                
                # Check for background tasks
                background_tasks = await self._identify_background_tasks(
                    analysis, brain_response
                )
                if background_tasks:
                    formatted_response["background_tasks"] = background_tasks
                
                logger.info("Brain processing completed", 
                           processing_time=processing_time,
                           response_type=brain_response.response_type.value,
                           actions_count=len(brain_response.actions),
                           confidence=brain_response.confidence)
                
                return formatted_response
                
            except Exception as e:
                processing_time = (datetime.now() - start_time).total_seconds()
                logger.error("Brain processing failed", 
                           error=str(e),
                           processing_time=processing_time)
                
                # Return error response
                return {
                    "response": f"I apologize, but I encountered an error while processing your request: {str(e)}",
                    "response_type": ResponseType.TEXT_RESPONSE.value,
                    "actions": [],
                    "tool_calls": [],
                    "context": context,
                    "confidence": 0.0,
                    "processing_time": processing_time,
                    "conversation_id": session_id,
                    "metadata": {"error": str(e)},
                    "error": True
                }
    
    async def _initialize_ai_client(self) -> None:
        """Initialize AI client (OpenAI/Anthropic)."""
        try:
            if settings.ai.openai_api_key:
                # Initialize OpenAI client
                import openai
                self.ai_client = openai.AsyncOpenAI(
                    api_key=settings.ai.openai_api_key
                )
                logger.info("OpenAI client initialized")
            
            elif settings.ai.anthropic_api_key:
                # Initialize Anthropic client
                import anthropic
                self.ai_client = anthropic.AsyncAnthropic(
                    api_key=settings.ai.anthropic_api_key
                )
                logger.info("Anthropic client initialized")
            
            else:
                logger.warning("No AI API keys configured")
                
        except Exception as e:
            logger.error("Failed to initialize AI client", error=str(e))
            raise
    
    async def _load_system_prompts(self) -> None:
        """Load system prompts and instructions."""
        self.system_prompt = """
        You are Xarvis, an advanced AI assistant inspired by Jarvis from Iron Man.
        You are intelligent, helpful, and capable of handling complex tasks.
        
        Key characteristics:
        - Professional but warm personality
        - Proactive and insightful
        - Capable of complex reasoning and problem-solving
        - Can use tools and execute actions
        - Maintain conversation context and memory
        - Provide clear, actionable responses
        
        You have access to various tools and systems:
        - Memory system for storing and retrieving information
        - Web search capabilities
        - Code generation and execution
        - System control and monitoring
        - Hardware integration
        
        Always strive to be helpful while being safe and ethical.
        """
        
        logger.info("System prompts loaded")
    
    async def _analyze_input(
        self,
        input_text: str,
        context: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Use LLM to intelligently analyze input and determine response strategy.
        
        Args:
            input_text: User input text
            context: Request context
        
        Returns:
            Analysis results from LLM
        """
        analysis_prompt = f"""
        Analyze this user input and provide a JSON response with the following fields:
        
        Input: "{input_text}"
        Context: {json.dumps(context, default=str)}
        
        Analyze and return JSON with:
        {{
            "intent": "search|creation|information|action|conversation|system_control",
            "requires_tools": true/false,
            "tool_suggestions": ["tool_name1", "tool_name2"],
            "requires_search": true/false,
            "search_type": "web|documentation|code|none",
            "requires_memory_lookup": true/false,
            "should_store_in_memory": true/false,
            "memory_importance": "high|medium|low",
            "complexity": "simple|moderate|complex",
            "response_type": "text|action|tool_call|audio|system",
            "urgency": "low|normal|high|critical",
            "expected_response_length": "short|medium|long",
            "conversation_flow": "continue|new_topic|end",
            "reasoning": "brief explanation of analysis"
        }}
        
        Consider:
        - Does this require external information or computation?
        - Is this worth storing in long-term memory?
        - What tools might be needed?
        - Is this part of ongoing conversation or new topic?
        """
        
        try:
            analysis_result = await self._query_llm_for_analysis(analysis_prompt)
            return analysis_result
        except Exception as e:
            logger.error("LLM analysis failed, falling back to simple analysis", error=str(e))
            return await self._simple_fallback_analysis(input_text, context)
    
    async def _generate_response(
        self,
        input_text: str,
        context: Dict[str, Any],
        analysis: Dict[str, Any],
        session_context: Dict[str, Any],
        processing_mode: ProcessingMode
    ) -> BrainResponse:
        """
        Generate response using AI model.
        
        Args:
            input_text: User input
            context: Request context
            analysis: Input analysis
            session_context: Session context
            processing_mode: Processing mode
        
        Returns:
            Generated brain response
        """
        try:
            # Build conversation messages
            messages = await self._build_conversation_messages(
                input_text, context, session_context, analysis
            )
            
            # Generate response using AI client
            if hasattr(self.ai_client, 'chat') and hasattr(self.ai_client.chat, 'completions'):
                # OpenAI
                response = await self.ai_client.chat.completions.create(
                    model=settings.ai.openai_model,
                    messages=messages,
                    max_tokens=2000,
                    temperature=0.7
                )
                content = response.choices[0].message.content
            
            elif hasattr(self.ai_client, 'messages'):
                # Anthropic
                response = await self.ai_client.messages.create(
                    model=settings.ai.anthropic_model,
                    max_tokens=2000,
                    temperature=0.7,
                    messages=messages
                )
                content = response.content[0].text
            
            else:
                # Fallback response
                content = await self._generate_fallback_response(input_text, analysis)
            
            # Parse response for actions and tool calls
            actions = await self._extract_actions_from_response(content)
            tool_calls = await self._extract_tool_calls_from_response(content)
            
            # Determine response type
            response_type = await self._determine_response_type(content, actions, tool_calls)
            
            # Calculate confidence
            confidence = await self._calculate_response_confidence(
                analysis, content, actions, tool_calls
            )
            
            return BrainResponse(
                response_id=str(uuid.uuid4()),
                response_type=response_type,
                content=content,
                actions=actions,
                tool_calls=tool_calls,
                context=context,
                confidence=confidence,
                processing_time=0.0,  # Will be calculated by caller
                metadata={
                    "model_used": settings.ai.openai_model or settings.ai.anthropic_model,
                    "processing_mode": processing_mode.value,
                    "analysis": analysis
                }
            )
            
        except Exception as e:
            logger.error("Failed to generate AI response", error=str(e))
            
            # Return fallback response
            fallback_content = await self._generate_fallback_response(input_text, analysis)
            
            return BrainResponse(
                response_id=str(uuid.uuid4()),
                response_type=ResponseType.TEXT_RESPONSE,
                content=fallback_content,
                actions=[],
                tool_calls=[],
                context=context,
                confidence=0.5,
                processing_time=0.0,
                metadata={"fallback": True, "error": str(e)}
            )
    
    async def _get_session_context(self, session_id: str) -> Dict[str, Any]:
        """Get or create session context."""
        if session_id not in self.active_sessions:
            self.active_sessions[session_id] = {
                "created_at": datetime.now().isoformat(),
                "last_activity": datetime.now().isoformat(),
                "message_count": 0,
                "context": {}
            }
        
        return self.active_sessions[session_id]
    
    async def _update_session_context(
        self,
        session_id: str,
        updates: Dict[str, Any]
    ) -> None:
        """Update session context."""
        if session_id in self.active_sessions:
            self.active_sessions[session_id].update(updates)
            self.active_sessions[session_id]["last_activity"] = datetime.now().isoformat()
            self.active_sessions[session_id]["message_count"] += 1
    
    async def _store_interaction(
        self,
        session_id: str,
        input_text: str,
        response: BrainResponse,
        correlation_id: str,
        importance_level: str = "medium"
    ) -> None:
        """Store interaction in memory system with importance weighting."""
        if self.memory_system:
            try:
                await self.memory_system.store_interaction(
                    session_id=session_id,
                    user_input=input_text,
                    assistant_response=response.content,
                    importance_level=importance_level,
                    metadata={
                        "correlation_id": correlation_id,
                        "response_type": response.response_type.value,
                        "confidence": response.confidence,
                        "actions": response.actions,
                        "timestamp": datetime.now().isoformat(),
                        "memory_importance": importance_level
                    }
                )
            except Exception as e:
                logger.error("Failed to store interaction", error=str(e))
    
    # Helper methods for LLM-based analysis
    async def _query_llm_for_analysis(self, prompt: str) -> Dict[str, Any]:
        """Query LLM for intelligent analysis."""
        try:
            messages = [
                {"role": "system", "content": "You are an AI analysis assistant. Respond only with valid JSON."},
                {"role": "user", "content": prompt}
            ]
            
            if hasattr(self.ai_client, 'chat') and hasattr(self.ai_client.chat, 'completions'):
                # OpenAI
                response = await self.ai_client.chat.completions.create(
                    model="gpt-3.5-turbo",  # Use faster model for analysis
                    messages=messages,
                    max_tokens=500,
                    temperature=0.3  # Lower temperature for consistent analysis
                )
                content = response.choices[0].message.content
            
            elif hasattr(self.ai_client, 'messages'):
                # Anthropic
                response = await self.ai_client.messages.create(
                    model="claude-3-haiku-20240307",  # Use faster model for analysis
                    max_tokens=500,
                    temperature=0.3,
                    messages=messages
                )
                content = response.content[0].text
            else:
                raise Exception("No AI client available")
            
            # Parse JSON response
            return json.loads(content)
            
        except json.JSONDecodeError as e:
            logger.error("Failed to parse LLM analysis JSON", error=str(e))
            raise
        except Exception as e:
            logger.error("LLM analysis query failed", error=str(e))
            raise
    
    async def _simple_fallback_analysis(self, input_text: str, context: Dict[str, Any]) -> Dict[str, Any]:
        """Simple fallback analysis when LLM fails."""
        text_lower = input_text.lower()
        
        # Simple keyword-based analysis as fallback
        requires_tools = any(word in text_lower for word in [
            "calculate", "compute", "run", "execute", "generate", "create", "search"
        ])
        
        requires_search = any(word in text_lower for word in [
            "search", "find", "latest", "current", "news", "look up"
        ])
        
        # Default to storing medium-length, informative conversations
        should_store = len(input_text) > 10 and not any(word in text_lower for word in [
            "hello", "hi", "bye", "thanks", "ok", "yes", "no"
        ])
        
        return {
            "intent": "conversation",
            "requires_tools": requires_tools,
            "tool_suggestions": [],
            "requires_search": requires_search,
            "search_type": "web" if requires_search else "none",
            "requires_memory_lookup": False,
            "should_store_in_memory": should_store,
            "memory_importance": "medium" if should_store else "low",
            "complexity": "moderate",
            "response_type": "text",
            "urgency": "normal",
            "expected_response_length": "medium",
            "conversation_flow": "continue",
            "reasoning": "Fallback analysis due to LLM unavailability"
        }
    
    async def _build_conversation_messages(
        self,
        input_text: str,
        context: Dict[str, Any],
        session_context: Dict[str, Any],
        analysis: Dict[str, Any]
    ) -> List[Dict[str, str]]:
        """Build conversation messages for AI model."""
        messages = [
            {"role": "system", "content": self.system_prompt}
        ]
        
        # Add conversation history if available
        history = context.get("conversation_history", [])
        for interaction in history[-5:]:  # Last 5 interactions
            messages.extend([
                {"role": "user", "content": interaction.get("user_input", "")},
                {"role": "assistant", "content": interaction.get("assistant_response", "")}
            ])
        
        # Add current input
        messages.append({"role": "user", "content": input_text})
        
        return messages
    
    async def _generate_fallback_response(
        self,
        input_text: str,
        analysis: Dict[str, Any]
    ) -> str:
        """Generate fallback response when AI is unavailable."""
        return f"I understand you're asking about '{input_text}'. I'm currently experiencing some technical difficulties, but I'm working to resolve them. Please try again in a moment."
    
    async def _extract_actions_from_response(self, content: str) -> List[Dict[str, Any]]:
        """Extract actions from AI response."""
        # Placeholder - would parse response for action indicators
        return []
    
    async def _extract_tool_calls_from_response(self, content: str) -> List[Dict[str, Any]]:
        """Extract tool calls from AI response."""
        # Placeholder - would parse response for tool call indicators
        return []
    
    async def _determine_response_type(
        self,
        content: str,
        actions: List[Dict[str, Any]],
        tool_calls: List[Dict[str, Any]]
    ) -> ResponseType:
        """Determine the type of response."""
        if tool_calls:
            return ResponseType.TOOL_CALL
        elif actions:
            return ResponseType.ACTION_SEQUENCE
        else:
            return ResponseType.TEXT_RESPONSE
    
    async def _calculate_response_confidence(
        self,
        analysis: Dict[str, Any],
        content: str,
        actions: List[Dict[str, Any]],
        tool_calls: List[Dict[str, Any]]
    ) -> float:
        """Calculate confidence score for response."""
        base_confidence = 0.8
        
        # Adjust based on analysis
        if analysis.get("complexity") == "simple":
            base_confidence += 0.1
        elif analysis.get("complexity") == "complex":
            base_confidence -= 0.1
        
        return max(0.1, min(1.0, base_confidence))
    
    def _needs_audio_response(
        self,
        analysis: Dict[str, Any],
        response: BrainResponse
    ) -> bool:
        """Check if response needs audio generation."""
        return True  # Default to always generating audio
    
    async def _get_voice_settings(self, context: Dict[str, Any]) -> Dict[str, Any]:
        """Get voice settings for audio response."""
        return {
            "voice": "default",
            "speed": 1.0,
            "pitch": 1.0,
            "volume": 0.8
        }
    
    async def _identify_background_tasks(
        self,
        analysis: Dict[str, Any],
        response: BrainResponse
    ) -> List[Dict[str, Any]]:
        """Identify any background tasks needed."""
        tasks = []
        
        if analysis.get("requires_search"):
            tasks.append({
                "name": "web_search_task",
                "args": [response.content],
                "priority": "normal"
            })
        
        return tasks
    
    async def health_check(self) -> Dict[str, Any]:
        """Perform brain health check."""
        return {
            "status": "healthy",
            "ai_client": self.ai_client is not None,
            "active_sessions": len(self.active_sessions),
            "memory_system": self.memory_system is not None,
            "tool_registry": self.tool_registry is not None
        }
    
    async def process_stream(
        self, 
        input_data: str, 
        context: Dict[str, Any], 
        user_id: str,
        session_id: str,
        correlation_id: str
    ) -> AsyncGenerator[Dict[str, Any], None]:
        """
        Process input with streaming response generation.
        
        Args:
            input_data: User input text
            context: Request context
            user_id: User identifier
            session_id: Session identifier  
            correlation_id: Correlation ID for tracking
        
        Yields:
            Streaming response chunks
        """
        with with_correlation_id(correlation_id):
            logger.info("Processing streaming input", 
                       user_id=user_id, 
                       session_id=session_id)
            
            try:
                # Yield initial processing status
                yield {
                    "type": "status",
                    "message": "Processing your request...",
                    "correlation_id": correlation_id
                }
                
                # Step 1: Analyze input
                yield {
                    "type": "status", 
                    "message": "Analyzing input...",
                    "correlation_id": correlation_id
                }
                
                analysis = await self._analyze_input_intelligent(input_data, context)
                
                # Step 2: Retrieve context if needed
                if analysis.get("needs_context"):
                    yield {
                        "type": "status",
                        "message": "Retrieving relevant context...", 
                        "correlation_id": correlation_id
                    }
                    
                    session_context = await self._get_session_context(session_id, user_id)
                else:
                    session_context = {}
                
                # Step 3: Execute tools if needed
                tool_results = []
                if analysis.get("tool_calls"):
                    yield {
                        "type": "status",
                        "message": "Executing tools...",
                        "correlation_id": correlation_id
                    }
                    
                    for tool_call in analysis["tool_calls"]:
                        tool_result = await self._execute_tool(tool_call, context)
                        tool_results.append(tool_result)
                        
                        yield {
                            "type": "tool_result",
                            "tool": tool_call.get("name"),
                            "result": tool_result,
                            "correlation_id": correlation_id
                        }
                
                # Step 4: Stream AI response generation
                yield {
                    "type": "status",
                    "message": "Generating response...",
                    "correlation_id": correlation_id
                }
                
                # Build conversation messages for streaming
                messages = await self._build_conversation_messages(
                    input_data, context, session_context, analysis
                )
                
                # Add tool results to context
                if tool_results:
                    tool_context = "\n\nTool Results:\n" + "\n".join([
                        f"- {result.get('tool', 'Unknown')}: {result.get('result', 'No result')}"
                        for result in tool_results
                    ])
                    messages.append({
                        "role": "user",
                        "content": f"Based on the tool results: {tool_context}"
                    })
                
                # Stream response from AI
                full_response = ""
                
                try:
                    if hasattr(self.ai_client, 'chat') and hasattr(self.ai_client.chat, 'completions'):
                        # OpenAI streaming
                        stream = await self.ai_client.chat.completions.create(
                            model=settings.ai.openai_model,
                            messages=messages,
                            max_tokens=2000,
                            temperature=0.7,
                            stream=True
                        )
                        
                        async for chunk in stream:
                            if chunk.choices[0].delta.content:
                                content = chunk.choices[0].delta.content
                                full_response += content
                                
                                yield {
                                    "type": "content",
                                    "content": content,
                                    "correlation_id": correlation_id
                                }
                    
                    elif hasattr(self.ai_client, 'messages'):
                        # Anthropic - simulate streaming for now
                        response = await self.ai_client.messages.create(
                            model=settings.ai.anthropic_model,
                            max_tokens=2000,
                            temperature=0.7,
                            messages=messages
                        )
                        
                        content = response.content[0].text
                        full_response = content
                        
                        # Simulate streaming by chunking response
                        words = content.split()
                        chunk_size = 3  # Send 3 words at a time
                        
                        for i in range(0, len(words), chunk_size):
                            chunk = " ".join(words[i:i+chunk_size])
                            if i + chunk_size < len(words):
                                chunk += " "
                            
                            yield {
                                "type": "content", 
                                "content": chunk,
                                "correlation_id": correlation_id
                            }
                            
                            # Small delay for realistic streaming
                            await asyncio.sleep(0.1)
                    
                    else:
                        # Fallback response
                        full_response = "I'm processing your request..."
                        yield {
                            "type": "content",
                            "content": full_response,
                            "correlation_id": correlation_id
                        }
                
                except Exception as ai_error:
                    logger.error("AI streaming failed", error=str(ai_error))
                    full_response = "I apologize, but I'm having trouble generating a response right now."
                    yield {
                        "type": "content",
                        "content": full_response,
                        "correlation_id": correlation_id
                    }
                
                # Step 5: Store memory if important
                if analysis.get("memory_importance", "low") in ["medium", "high"]:
                    yield {
                        "type": "status",
                        "message": "Storing conversation...",
                        "correlation_id": correlation_id
                    }
                    
                    await self._store_conversation_memory(
                        user_id, session_id, input_data, full_response, 
                        analysis["memory_importance"], context
                    )
                
                # Final completion message
                yield {
                    "type": "complete",
                    "full_response": full_response,
                    "actions": analysis.get("actions", []),
                    "conversation_id": session_id,
                    "correlation_id": correlation_id
                }
                
            except Exception as e:
                logger.error("Streaming processing failed", error=str(e))
                yield {
                    "type": "error",
                    "error": str(e), 
                    "correlation_id": correlation_id
                }
