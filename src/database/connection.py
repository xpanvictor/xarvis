"""
Database Connection - Database connection and session management.
Provides SQLAlchemy database connection utilities and session handling.
"""

from typing import Generator, Optional
from contextlib import contextmanager
from sqlalchemy import create_engine, event
from sqlalchemy.orm import sessionmaker, Session
from sqlalchemy.pool import QueuePool
import asyncio
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine, async_sessionmaker

from src.config import settings
from src.utils.logging import get_logger
from .models import Base


logger = get_logger(__name__)

# Database engines (sync and async)
sync_engine = None
async_engine = None

# Session factories
SessionLocal = None
AsyncSessionLocal = None


def init_database() -> None:
    """Initialize database connections and create tables."""
    global sync_engine, async_engine, SessionLocal, AsyncSessionLocal
    
    logger.info("Initializing database connections")
    
    try:
        # Create synchronous engine
        sync_engine = create_engine(
            settings.database.url,
            poolclass=QueuePool,
            pool_size=settings.database.pool_size,
            max_overflow=settings.database.max_overflow,
            pool_timeout=settings.database.pool_timeout,
            pool_recycle=settings.database.pool_recycle,
            echo=settings.database.echo,
            future=True
        )
        
        # Create session factory
        SessionLocal = sessionmaker(
            bind=sync_engine,
            autocommit=False,
            autoflush=False,
            expire_on_commit=False
        )
        
        # Create asynchronous engine if async URL is provided
        if hasattr(settings.database, 'async_url') and settings.database.async_url:
            async_engine = create_async_engine(
                settings.database.async_url,
                pool_size=settings.database.pool_size,
                max_overflow=settings.database.max_overflow,
                pool_timeout=settings.database.pool_timeout,
                pool_recycle=settings.database.pool_recycle,
                echo=settings.database.echo,
                future=True
            )
            
            # Create async session factory
            AsyncSessionLocal = async_sessionmaker(
                bind=async_engine,
                class_=AsyncSession,
                autocommit=False,
                autoflush=False,
                expire_on_commit=False
            )
        
        # Create all tables
        Base.metadata.create_all(bind=sync_engine)
        
        # Add connection event listeners
        _add_connection_listeners()
        
        logger.info("Database initialization completed successfully")
        
    except Exception as e:
        logger.error("Database initialization failed", error=str(e))
        raise


def close_database() -> None:
    """Close database connections."""
    global sync_engine, async_engine
    
    logger.info("Closing database connections")
    
    try:
        if sync_engine:
            sync_engine.dispose()
            sync_engine = None
        
        if async_engine:
            asyncio.create_task(async_engine.dispose())
            async_engine = None
        
        logger.info("Database connections closed")
        
    except Exception as e:
        logger.error("Error closing database connections", error=str(e))


@contextmanager
def get_database_session() -> Generator[Session, None, None]:
    """
    Get database session with automatic cleanup.
    
    Yields:
        Database session
    """
    if SessionLocal is None:
        raise RuntimeError("Database not initialized. Call init_database() first.")
    
    session = SessionLocal()
    try:
        logger.debug("Database session created")
        yield session
        session.commit()
        logger.debug("Database session committed")
    except Exception as e:
        logger.error("Database session error", error=str(e))
        session.rollback()
        raise
    finally:
        session.close()
        logger.debug("Database session closed")


async def get_async_database_session() -> AsyncSession:
    """
    Get async database session.
    
    Returns:
        Async database session
    """
    if AsyncSessionLocal is None:
        raise RuntimeError("Async database not initialized or not configured.")
    
    return AsyncSessionLocal()


@contextmanager
def get_database_session_context():
    """Context manager for database sessions."""
    with get_database_session() as session:
        yield session


def get_session_factory():
    """Get session factory for dependency injection."""
    return SessionLocal


def get_async_session_factory():
    """Get async session factory for dependency injection.""" 
    return AsyncSessionLocal


def create_database_tables(engine=None) -> None:
    """
    Create all database tables.
    
    Args:
        engine: Database engine (uses default if not provided)
    """
    target_engine = engine or sync_engine
    
    if target_engine is None:
        raise RuntimeError("No database engine available")
    
    logger.info("Creating database tables")
    
    try:
        Base.metadata.create_all(bind=target_engine)
        logger.info("Database tables created successfully")
    except Exception as e:
        logger.error("Failed to create database tables", error=str(e))
        raise


def drop_database_tables(engine=None) -> None:
    """
    Drop all database tables.
    
    Args:
        engine: Database engine (uses default if not provided)
    """
    target_engine = engine or sync_engine
    
    if target_engine is None:
        raise RuntimeError("No database engine available")
    
    logger.warning("Dropping all database tables")
    
    try:
        Base.metadata.drop_all(bind=target_engine)
        logger.info("Database tables dropped successfully")
    except Exception as e:
        logger.error("Failed to drop database tables", error=str(e))
        raise


def check_database_connection() -> bool:
    """
    Check if database connection is working.
    
    Returns:
        True if connection is working
    """
    try:
        with get_database_session() as session:
            # Execute a simple query to test connection
            result = session.execute("SELECT 1")
            result.fetchone()
            logger.debug("Database connection check successful")
            return True
    except Exception as e:
        logger.error("Database connection check failed", error=str(e))
        return False


async def check_async_database_connection() -> bool:
    """
    Check if async database connection is working.
    
    Returns:
        True if connection is working
    """
    if AsyncSessionLocal is None:
        return False
    
    try:
        async with AsyncSessionLocal() as session:
            # Execute a simple query to test connection
            result = await session.execute("SELECT 1")
            result.fetchone()
            logger.debug("Async database connection check successful")
            return True
    except Exception as e:
        logger.error("Async database connection check failed", error=str(e))
        return False


def get_database_info() -> dict:
    """
    Get database connection information.
    
    Returns:
        Database info dictionary
    """
    info = {
        "sync_engine_initialized": sync_engine is not None,
        "async_engine_initialized": async_engine is not None,
        "session_factory_initialized": SessionLocal is not None,
        "async_session_factory_initialized": AsyncSessionLocal is not None,
    }
    
    if sync_engine:
        info.update({
            "sync_engine_url": str(sync_engine.url).replace(sync_engine.url.password or "", "***"),
            "sync_pool_size": sync_engine.pool.size(),
            "sync_checked_out": sync_engine.pool.checkedout(),
        })
    
    if async_engine:
        info.update({
            "async_engine_url": str(async_engine.url).replace(async_engine.url.password or "", "***"),
            "async_pool_size": async_engine.pool.size(),
            "async_checked_out": async_engine.pool.checkedout(),
        })
    
    return info


def _add_connection_listeners() -> None:
    """Add database connection event listeners."""
    if sync_engine is None:
        return
    
    @event.listens_for(sync_engine, "connect")
    def receive_connect(dbapi_connection, connection_record):
        """Handle database connection events."""
        logger.debug("Database connection established")
        
        # Set connection-level settings if needed
        if settings.database.url.startswith("postgresql"):
            # PostgreSQL specific settings
            with dbapi_connection.cursor() as cursor:
                cursor.execute("SET timezone='UTC'")
    
    @event.listens_for(sync_engine, "disconnect")
    def receive_disconnect(dbapi_connection, connection_record):
        """Handle database disconnection events.""" 
        logger.debug("Database connection closed")
    
    @event.listens_for(sync_engine.pool, "connect")
    def receive_pool_connect(dbapi_connection, connection_record):
        """Handle connection pool events."""
        logger.debug("Connection added to pool")
    
    @event.listens_for(sync_engine.pool, "checkout")
    def receive_pool_checkout(dbapi_connection, connection_record, connection_proxy):
        """Handle connection checkout events."""
        logger.debug("Connection checked out from pool")
    
    @event.listens_for(sync_engine.pool, "checkin")
    def receive_pool_checkin(dbapi_connection, connection_record):
        """Handle connection checkin events."""
        logger.debug("Connection returned to pool")


# Dependency for FastAPI
def get_db() -> Generator[Session, None, None]:
    """
    FastAPI dependency for database sessions.
    
    Yields:
        Database session
    """
    with get_database_session() as session:
        yield session


async def get_async_db() -> Generator[AsyncSession, None, None]:
    """
    FastAPI dependency for async database sessions.
    
    Yields:
        Async database session
    """
    if AsyncSessionLocal is None:
        raise RuntimeError("Async database not configured")
    
    async with AsyncSessionLocal() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise
        finally:
            await session.close()


# Export functions
__all__ = [
    "init_database",
    "close_database", 
    "get_database_session",
    "get_async_database_session",
    "get_database_session_context",
    "get_session_factory",
    "get_async_session_factory",
    "create_database_tables",
    "drop_database_tables",
    "check_database_connection",
    "check_async_database_connection",
    "get_database_info",
    "get_db",
    "get_async_db"
]
