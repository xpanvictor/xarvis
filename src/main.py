"""
Main application module for Xarvis AI Assistant System.
Initializes the FastAPI application and configures all components.
"""

from fastapi import FastAPI, HTTPException, Depends
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.gzip import GZipMiddleware
from fastapi.responses import JSONResponse
from contextlib import asynccontextmanager
import uvicorn

from src.config import settings
from src.utils.logging import setup_logging, get_logger
from src.backbone.pipeline_server import PipelineServer
from src.backbone.job_runner import JobRunner
from src.backbone.aggregator import Aggregator
from src.backbone.brain import Brain
from src.interfaces.audio_interface import AudioInterface
from src.interfaces.hardware_interface import HardwareInterface
from src.core.nlp import NLPProcessor
from src.core.memory import MemorySystem
from src.core.tools import ToolRegistry
from src.api.conversation import router as conversation_router


# Initialize logging
setup_logging(
    log_level=settings.logging.level,
    log_format=settings.logging.format,
    log_file=settings.logging.file
)

logger = get_logger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager."""
    logger.info("Starting Xarvis AI Assistant System", version="1.0.0")
    
    # Initialize core components
    try:
        # Initialize memory system
        await app.state.memory_system.initialize()
        logger.info("Memory system initialized")
        
        # Initialize brain
        await app.state.brain.initialize()
        logger.info("Brain initialized")
        
        # Initialize interfaces
        await app.state.audio_interface.initialize()
        await app.state.hardware_interface.initialize()
        logger.info("Interfaces initialized")
        
        # Start pipeline server
        await app.state.pipeline_server.start()
        logger.info("Pipeline server started")
        
        logger.info("🤖 Xarvis is now online and ready to assist!")
        
    except Exception as e:
        logger.error("Failed to initialize Xarvis", error=str(e))
        raise
    
    yield
    
    # Cleanup
    logger.info("Shutting down Xarvis...")
    try:
        await app.state.pipeline_server.stop()
        await app.state.audio_interface.cleanup()
        await app.state.hardware_interface.cleanup()
        logger.info("✅ Xarvis shutdown complete")
    except Exception as e:
        logger.error("Error during shutdown", error=str(e))


# Create FastAPI application
app = FastAPI(
    title="Xarvis AI Assistant",
    description="Advanced AI assistant system inspired by Jarvis from Iron Man",
    version="1.0.0",
    docs_url="/docs",
    redoc_url="/redoc",
    lifespan=lifespan
)

# Add middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Configure appropriately for production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.add_middleware(GZipMiddleware, minimum_size=1000)

# Include API routers
app.include_router(conversation_router)


# Initialize core components
@app.on_event("startup")
async def startup_event():
    """Initialize application components on startup."""
    
    # Core systems
    app.state.memory_system = MemorySystem()
    app.state.nlp_processor = NLPProcessor()
    app.state.tool_registry = ToolRegistry()
    app.state.brain = Brain(
        memory_system=app.state.memory_system,
        nlp_processor=app.state.nlp_processor,
        tool_registry=app.state.tool_registry
    )
    
    # Backbone components
    app.state.job_runner = JobRunner()
    app.state.aggregator = Aggregator(memory_system=app.state.memory_system)
    app.state.pipeline_server = PipelineServer(
        brain=app.state.brain,
        job_runner=app.state.job_runner,
        aggregator=app.state.aggregator
    )
    
    # Interfaces
    app.state.audio_interface = AudioInterface()
    app.state.hardware_interface = HardwareInterface()


# Health check endpoint
@app.get("/health")
async def health_check():
    """System health check endpoint."""
    try:
        # Check core components
        memory_status = await app.state.memory_system.health_check()
        brain_status = await app.state.brain.health_check()
        
        return JSONResponse({
            "status": "healthy",
            "timestamp": app.state.pipeline_server.get_timestamp(),
            "version": "1.0.0",
            "components": {
                "memory": memory_status,
                "brain": brain_status,
                "pipeline": "healthy",
                "interfaces": "healthy"
            }
        })
    except Exception as e:
        logger.error("Health check failed", error=str(e))
        return JSONResponse(
            {"status": "unhealthy", "error": str(e)}, 
            status_code=503
        )


# Main conversation endpoint
@app.post("/chat")
async def chat(message: dict):
    """
    Main chat endpoint for processing user input.
    
    Args:
        message: Dictionary containing user input and metadata
    
    Returns:
        Response from the AI system
    """
    try:
        user_input = message.get("text", "")
        audio_data = message.get("audio", None)
        metadata = message.get("metadata", {})
        
        if not user_input and not audio_data:
            raise HTTPException(status_code=400, detail="No input provided")
        
        # Process through pipeline
        response = await app.state.pipeline_server.process_input(
            text=user_input,
            audio=audio_data,
            metadata=metadata
        )
        
        return JSONResponse(response)
        
    except Exception as e:
        logger.error("Chat processing failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


# Audio processing endpoints
@app.post("/audio/process")
async def process_audio(audio_data: dict):
    """Process audio input from hardware."""
    try:
        processed = await app.state.audio_interface.process_input(audio_data)
        return JSONResponse(processed)
    except Exception as e:
        logger.error("Audio processing failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/audio/output/{response_id}")
async def get_audio_output(response_id: str):
    """Get audio output for hardware playback."""
    try:
        audio_output = await app.state.audio_interface.get_output(response_id)
        return JSONResponse(audio_output)
    except Exception as e:
        logger.error("Audio output failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


# System management endpoints
@app.post("/system/task")
async def submit_task(task: dict):
    """Submit a background task for processing."""
    try:
        task_id = await app.state.job_runner.submit_task(task)
        return JSONResponse({"task_id": task_id})
    except Exception as e:
        logger.error("Task submission failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/system/task/{task_id}")
async def get_task_status(task_id: str):
    """Get the status of a background task."""
    try:
        status = await app.state.job_runner.get_task_status(task_id)
        return JSONResponse(status)
    except Exception as e:
        logger.error("Task status retrieval failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


# Memory management endpoints
@app.post("/memory/store")
async def store_memory(memory_data: dict):
    """Store information in the memory system."""
    try:
        memory_id = await app.state.memory_system.store(memory_data)
        return JSONResponse({"memory_id": memory_id})
    except Exception as e:
        logger.error("Memory storage failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/memory/search")
async def search_memory(query: str, limit: int = 10):
    """Search the memory system."""
    try:
        results = await app.state.memory_system.search(query, limit=limit)
        return JSONResponse(results)
    except Exception as e:
        logger.error("Memory search failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


if __name__ == "__main__":
    uvicorn.run(
        "src.main:app",
        host=settings.api.host,
        port=settings.api.port,
        reload=settings.debug,
        workers=1 if settings.debug else settings.api.workers
    )
