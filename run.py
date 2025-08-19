#!/usr/bin/env python3
"""
Jarvis System Startup Script
Comprehensive startup and management script for the Jarvis AI assistant system.
"""

import asyncio
import sys
import signal
import time
from pathlib import Path
from typing import Optional

# Add src to path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from src.config import settings
from src.utils.logging import get_logger, setup_logging
from src.database.connection import init_database, close_database, check_database_connection

# Initialize logging first
setup_logging()
logger = get_logger(__name__)


class JarvisSystem:
    """Main system orchestrator for Jarvis."""
    
    def __init__(self):
        self.components = {}
        self.is_running = False
        self.shutdown_event = asyncio.Event()
        
    async def initialize(self):
        """Initialize all system components."""
        logger.info("🚀 Starting Jarvis AI Assistant System")
        logger.info("=" * 60)
        
        try:
            # Initialize database
            logger.info("📊 Initializing database...")
            init_database()
            
            if not check_database_connection():
                raise Exception("Database connection failed")
            logger.info("✅ Database initialized successfully")
            
            # Initialize core components
            await self._initialize_core_components()
            
            # Initialize backbone components
            await self._initialize_backbone_components()
            
            # Initialize interface components
            await self._initialize_interface_components()
            
            logger.info("✅ All components initialized successfully")
            logger.info("🎉 Jarvis system is ready!")
            
        except Exception as e:
            logger.error("💥 System initialization failed", error=str(e))
            raise
    
    async def _initialize_core_components(self):
        """Initialize core system components."""
        logger.info("🧠 Initializing core components...")
        
        try:
            # Memory System
            from src.core.memory_system import MemorySystem
            memory_system = MemorySystem()
            await memory_system.initialize()
            self.components['memory'] = memory_system
            logger.info("  ✓ Memory system initialized")
            
            # NLP Processor
            from src.core.nlp_processor import NLPProcessor
            nlp_processor = NLPProcessor()
            await nlp_processor.initialize()
            self.components['nlp'] = nlp_processor
            logger.info("  ✓ NLP processor initialized")
            
            # Tool Registry
            from src.core.tool_registry import ToolRegistry
            tool_registry = ToolRegistry()
            await tool_registry.initialize()
            self.components['tools'] = tool_registry
            logger.info("  ✓ Tool registry initialized")
            
            logger.info("✅ Core components initialized")
            
        except Exception as e:
            logger.error("💥 Core components initialization failed", error=str(e))
            raise
    
    async def _initialize_backbone_components(self):
        """Initialize backbone system components."""
        logger.info("🏗️ Initializing backbone components...")
        
        try:
            # Pipeline Server
            from src.backbone.pipeline_server import PipelineServer
            pipeline_server = PipelineServer()
            await pipeline_server.initialize()
            self.components['pipeline'] = pipeline_server
            logger.info("  ✓ Pipeline server initialized")
            
            # Job Runner (Celery integration)
            from src.backbone.job_runner import JobRunner
            job_runner = JobRunner()
            await job_runner.initialize()
            self.components['jobs'] = job_runner
            logger.info("  ✓ Job runner initialized")
            
            # Data Aggregator
            from src.backbone.aggregator import Aggregator
            aggregator = Aggregator()
            await aggregator.initialize()
            self.components['aggregator'] = aggregator
            logger.info("  ✓ Data aggregator initialized")
            
            # AI Brain
            from src.backbone.brain import Brain
            brain = Brain()
            await brain.initialize()
            self.components['brain'] = brain
            logger.info("  ✓ AI brain initialized")
            
            logger.info("✅ Backbone components initialized")
            
        except Exception as e:
            logger.error("💥 Backbone components initialization failed", error=str(e))
            raise
    
    async def _initialize_interface_components(self):
        """Initialize interface components."""
        logger.info("🎤 Initializing interface components...")
        
        try:
            # Audio Interface
            from src.interfaces.audio_interface import AudioInterface
            audio_interface = AudioInterface()
            await audio_interface.initialize()
            self.components['audio'] = audio_interface
            logger.info("  ✓ Audio interface initialized")
            
            # Hardware Interface
            from src.interfaces.hardware_interface import HardwareInterface
            hardware_interface = HardwareInterface()
            await hardware_interface.initialize()
            self.components['hardware'] = hardware_interface
            logger.info("  ✓ Hardware interface initialized")
            
            logger.info("✅ Interface components initialized")
            
        except Exception as e:
            logger.error("💥 Interface components initialization failed", error=str(e))
            raise
    
    async def start_services(self):
        """Start all background services."""
        logger.info("🔄 Starting background services...")
        
        try:
            # Start FastAPI server (in background)
            await self._start_api_server()
            
            # Start Celery workers (if not running externally)
            if settings.celery.start_worker:
                await self._start_celery_worker()
            
            # Start monitoring tasks
            await self._start_monitoring()
            
            logger.info("✅ All services started")
            
        except Exception as e:
            logger.error("💥 Service startup failed", error=str(e))
            raise
    
    async def _start_api_server(self):
        """Start FastAPI server."""
        try:
            import uvicorn
            from src.main import app
            
            config = uvicorn.Config(
                app,
                host=settings.server.host,
                port=settings.server.port,
                log_level="info",
                access_log=True
            )
            
            server = uvicorn.Server(config)
            self.components['api_server'] = server
            
            # Start server in background
            asyncio.create_task(server.serve())
            logger.info(f"  ✓ API server started on {settings.server.host}:{settings.server.port}")
            
        except Exception as e:
            logger.error("API server startup failed", error=str(e))
            raise
    
    async def _start_celery_worker(self):
        """Start Celery worker."""
        try:
            import subprocess
            
            # Start Celery worker process
            celery_cmd = [
                sys.executable, "-m", "celery",
                "-A", "celery_app",
                "worker",
                "--loglevel=info",
                "--concurrency=4"
            ]
            
            process = subprocess.Popen(celery_cmd)
            self.components['celery_worker'] = process
            
            logger.info("  ✓ Celery worker started")
            
        except Exception as e:
            logger.error("Celery worker startup failed", error=str(e))
            raise
    
    async def _start_monitoring(self):
        """Start system monitoring."""
        try:
            # Create monitoring task
            async def monitor_system():
                while self.is_running:
                    try:
                        # Perform health checks
                        await self._health_check()
                        await asyncio.sleep(60)  # Check every minute
                    except Exception as e:
                        logger.error("Monitoring error", error=str(e))
                        await asyncio.sleep(60)
            
            # Start monitoring task
            asyncio.create_task(monitor_system())
            logger.info("  ✓ System monitoring started")
            
        except Exception as e:
            logger.error("Monitoring startup failed", error=str(e))
            raise
    
    async def _health_check(self):
        """Perform system health check."""
        try:
            # Check database
            if not check_database_connection():
                logger.warning("Database health check failed")
            
            # Check components
            for name, component in self.components.items():
                if hasattr(component, 'health_check'):
                    try:
                        await component.health_check()
                    except Exception as e:
                        logger.warning(f"Component {name} health check failed", error=str(e))
            
        except Exception as e:
            logger.error("Health check failed", error=str(e))
    
    async def run(self):
        """Run the main system loop."""
        self.is_running = True
        logger.info("🏃 Jarvis system running...")
        
        try:
            # Wait for shutdown signal
            await self.shutdown_event.wait()
            
        except KeyboardInterrupt:
            logger.info("Received keyboard interrupt")
        except Exception as e:
            logger.error("System runtime error", error=str(e))
        finally:
            await self.shutdown()
    
    async def shutdown(self):
        """Shutdown all system components."""
        logger.info("🛑 Shutting down Jarvis system...")
        self.is_running = False
        
        try:
            # Shutdown interface components
            if 'hardware' in self.components:
                await self.components['hardware'].cleanup()
                logger.info("  ✓ Hardware interface shut down")
            
            if 'audio' in self.components:
                await self.components['audio'].cleanup()
                logger.info("  ✓ Audio interface shut down")
            
            # Shutdown backbone components
            if 'brain' in self.components:
                await self.components['brain'].cleanup()
                logger.info("  ✓ AI brain shut down")
            
            if 'aggregator' in self.components:
                await self.components['aggregator'].cleanup()
                logger.info("  ✓ Data aggregator shut down")
            
            if 'jobs' in self.components:
                await self.components['jobs'].cleanup()
                logger.info("  ✓ Job runner shut down")
            
            if 'pipeline' in self.components:
                await self.components['pipeline'].cleanup()
                logger.info("  ✓ Pipeline server shut down")
            
            # Shutdown core components
            if 'tools' in self.components:
                await self.components['tools'].cleanup()
                logger.info("  ✓ Tool registry shut down")
            
            if 'nlp' in self.components:
                await self.components['nlp'].cleanup()
                logger.info("  ✓ NLP processor shut down")
            
            if 'memory' in self.components:
                await self.components['memory'].cleanup()
                logger.info("  ✓ Memory system shut down")
            
            # Shutdown external services
            if 'celery_worker' in self.components:
                self.components['celery_worker'].terminate()
                logger.info("  ✓ Celery worker shut down")
            
            # Close database
            close_database()
            logger.info("  ✓ Database connections closed")
            
            logger.info("✅ Jarvis system shut down complete")
            
        except Exception as e:
            logger.error("Shutdown error", error=str(e))


async def main():
    """Main entry point."""
    system = JarvisSystem()
    
    # Setup signal handlers
    def signal_handler(signum, frame):
        logger.info(f"Received signal {signum}")
        asyncio.create_task(system.shutdown_event.set())
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    try:
        # Initialize system
        await system.initialize()
        
        # Start services
        await system.start_services()
        
        # Run main loop
        await system.run()
        
    except Exception as e:
        logger.error("System startup failed", error=str(e))
        sys.exit(1)


def start_celery_worker():
    """Start Celery worker process."""
    import subprocess
    import os
    
    logger.info("🔄 Starting Celery worker...")
    
    env = os.environ.copy()
    env['PYTHONPATH'] = str(Path(__file__).parent)
    
    cmd = [
        sys.executable, "-m", "celery",
        "-A", "celery_app",
        "worker",
        "--loglevel=info",
        "--concurrency=4",
        "--queues=brain,memory,search,audio,system"
    ]
    
    try:
        subprocess.run(cmd, env=env, check=True)
    except KeyboardInterrupt:
        logger.info("Celery worker stopped")
    except Exception as e:
        logger.error("Celery worker failed", error=str(e))
        sys.exit(1)


def start_celery_beat():
    """Start Celery beat scheduler."""
    import subprocess
    import os
    
    logger.info("⏰ Starting Celery beat scheduler...")
    
    env = os.environ.copy()
    env['PYTHONPATH'] = str(Path(__file__).parent)
    
    cmd = [
        sys.executable, "-m", "celery",
        "-A", "celery_app",
        "beat",
        "--loglevel=info"
    ]
    
    try:
        subprocess.run(cmd, env=env, check=True)
    except KeyboardInterrupt:
        logger.info("Celery beat stopped")
    except Exception as e:
        logger.error("Celery beat failed", error=str(e))
        sys.exit(1)


def start_flower_monitoring():
    """Start Flower monitoring for Celery."""
    import subprocess
    import os
    
    logger.info("🌸 Starting Flower monitoring...")
    
    env = os.environ.copy()
    env['PYTHONPATH'] = str(Path(__file__).parent)
    
    cmd = [
        sys.executable, "-m", "celery",
        "-A", "celery_app",
        "flower",
        "--port=5555"
    ]
    
    try:
        subprocess.run(cmd, env=env, check=True)
    except KeyboardInterrupt:
        logger.info("Flower monitoring stopped")
    except Exception as e:
        logger.error("Flower monitoring failed", error=str(e))
        sys.exit(1)


if __name__ == "__main__":
    if len(sys.argv) > 1:
        command = sys.argv[1]
        
        if command == "worker":
            start_celery_worker()
        elif command == "beat":
            start_celery_beat()
        elif command == "flower":
            start_flower_monitoring()
        else:
            print(f"Unknown command: {command}")
            print("Available commands: worker, beat, flower")
            sys.exit(1)
    else:
        # Start main system
        asyncio.run(main())
