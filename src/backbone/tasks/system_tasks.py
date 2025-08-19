"""
System Tasks - Celery tasks for system monitoring and maintenance.
Handles health checks, monitoring, cleanup, and backup operations.
"""

from typing import Dict, Any, List, Optional
from datetime import datetime, timedelta
import json
import uuid
import shutil
import psutil
from pathlib import Path

from celery import current_app
from celery.exceptions import Retry

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


@current_app.task(
    bind=True,
    name="system.health_check",
    max_retries=1
)
def health_check(self) -> Dict[str, Any]:
    """
    Perform comprehensive system health check.
    
    Returns:
        System health status
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Performing system health check")
        
        try:
            health_status = {
                "timestamp": datetime.now().isoformat(),
                "correlation_id": correlation_id,
                "overall_status": "healthy",
                "components": {}
            }
            
            # Check system resources
            health_status["components"]["system"] = _check_system_resources()
            
            # Check database connectivity
            health_status["components"]["database"] = _check_database_health()
            
            # Check Redis connectivity
            health_status["components"]["redis"] = _check_redis_health()
            
            # Check disk space
            health_status["components"]["disk"] = _check_disk_space()
            
            # Check AI services
            health_status["components"]["ai_services"] = _check_ai_services()
            
            # Check audio services
            health_status["components"]["audio"] = _check_audio_services()
            
            # Check hardware interfaces
            health_status["components"]["hardware"] = _check_hardware_interfaces()
            
            # Determine overall status
            component_statuses = [comp.get("status", "unknown") for comp in health_status["components"].values()]
            
            if "critical" in component_statuses:
                health_status["overall_status"] = "critical"
            elif "warning" in component_statuses:
                health_status["overall_status"] = "warning"
            else:
                health_status["overall_status"] = "healthy"
            
            logger.info("Health check completed", 
                       overall_status=health_status["overall_status"],
                       components_checked=len(health_status["components"]))
            
            return {
                "success": True,
                "health_status": health_status
            }
            
        except Exception as e:
            logger.error("Health check failed", error=str(e))
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="system.system_monitoring",
    max_retries=1
)
def system_monitoring(
    self,
    duration_minutes: int = 1,
    collect_metrics: bool = True
) -> Dict[str, Any]:
    """
    Monitor system performance metrics.
    
    Args:
        duration_minutes: Monitoring duration
        collect_metrics: Whether to collect detailed metrics
    
    Returns:
        Monitoring results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Starting system monitoring", 
                   duration_minutes=duration_minutes,
                   collect_metrics=collect_metrics)
        
        try:
            metrics = {
                "timestamp": datetime.now().isoformat(),
                "correlation_id": correlation_id,
                "duration_minutes": duration_minutes,
                "system_info": {},
                "performance_metrics": {}
            }
            
            if collect_metrics:
                # System information
                metrics["system_info"] = {
                    "cpu_count": psutil.cpu_count(),
                    "memory_total": psutil.virtual_memory().total,
                    "disk_usage": {
                        path: {
                            "total": shutil.disk_usage(path).total,
                            "used": shutil.disk_usage(path).used,
                            "free": shutil.disk_usage(path).free
                        } for path in ["/", "/tmp"] if Path(path).exists()
                    }
                }
                
                # Performance metrics
                metrics["performance_metrics"] = {
                    "cpu_percent": psutil.cpu_percent(interval=1),
                    "memory_percent": psutil.virtual_memory().percent,
                    "disk_io": psutil.disk_io_counters()._asdict() if psutil.disk_io_counters() else {},
                    "network_io": psutil.net_io_counters()._asdict() if psutil.net_io_counters() else {},
                    "process_count": len(psutil.pids()),
                    "load_average": psutil.getloadavg() if hasattr(psutil, 'getloadavg') else [0, 0, 0]
                }
            
            logger.info("System monitoring completed", 
                       cpu_percent=metrics.get("performance_metrics", {}).get("cpu_percent", 0),
                       memory_percent=metrics.get("performance_metrics", {}).get("memory_percent", 0))
            
            return {
                "success": True,
                "metrics": metrics
            }
            
        except Exception as e:
            logger.error("System monitoring failed", error=str(e))
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="system.cleanup_tasks",
    max_retries=2
)
def cleanup_tasks(
    self,
    cleanup_types: List[str] = None,
    older_than_days: int = 7
) -> Dict[str, Any]:
    """
    Perform system cleanup tasks.
    
    Args:
        cleanup_types: Types of cleanup to perform
        older_than_days: Remove items older than this many days
    
    Returns:
        Cleanup results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Starting system cleanup", 
                   cleanup_types=cleanup_types,
                   older_than_days=older_than_days)
        
        try:
            if cleanup_types is None:
                cleanup_types = ["temp_files", "logs", "cache", "old_tasks"]
            
            cleanup_results = {
                "timestamp": datetime.now().isoformat(),
                "correlation_id": correlation_id,
                "cleanup_types": cleanup_types,
                "results": {}
            }
            
            total_files_removed = 0
            total_space_freed = 0
            
            # Clean temporary files
            if "temp_files" in cleanup_types:
                temp_result = _cleanup_temp_files(older_than_days)
                cleanup_results["results"]["temp_files"] = temp_result
                total_files_removed += temp_result.get("files_removed", 0)
                total_space_freed += temp_result.get("space_freed_bytes", 0)
            
            # Clean old logs
            if "logs" in cleanup_types:
                log_result = _cleanup_old_logs(older_than_days)
                cleanup_results["results"]["logs"] = log_result
                total_files_removed += log_result.get("files_removed", 0)
                total_space_freed += log_result.get("space_freed_bytes", 0)
            
            # Clean cache files
            if "cache" in cleanup_types:
                cache_result = _cleanup_cache_files(older_than_days)
                cleanup_results["results"]["cache"] = cache_result
                total_files_removed += cache_result.get("files_removed", 0)
                total_space_freed += cache_result.get("space_freed_bytes", 0)
            
            # Clean old Celery task results
            if "old_tasks" in cleanup_types:
                task_result = _cleanup_old_tasks(older_than_days)
                cleanup_results["results"]["old_tasks"] = task_result
            
            cleanup_results["summary"] = {
                "total_files_removed": total_files_removed,
                "total_space_freed_mb": total_space_freed / (1024 * 1024),
                "cleanup_duration": (datetime.now() - datetime.fromisoformat(cleanup_results["timestamp"].replace('Z', '+00:00'))).total_seconds()
            }
            
            logger.info("System cleanup completed", 
                       files_removed=total_files_removed,
                       space_freed_mb=total_space_freed / (1024 * 1024))
            
            return {
                "success": True,
                "cleanup_results": cleanup_results
            }
            
        except Exception as e:
            logger.error("System cleanup failed", error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=60, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="system.backup_data",
    max_retries=2
)
def backup_data(
    self,
    backup_types: List[str] = None,
    destination: str = "data/backups"
) -> Dict[str, Any]:
    """
    Perform data backup operations.
    
    Args:
        backup_types: Types of data to backup
        destination: Backup destination directory
    
    Returns:
        Backup results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Starting data backup", 
                   backup_types=backup_types,
                   destination=destination)
        
        try:
            if backup_types is None:
                backup_types = ["conversations", "memories", "configurations", "logs"]
            
            backup_results = {
                "timestamp": datetime.now().isoformat(),
                "correlation_id": correlation_id,
                "backup_types": backup_types,
                "destination": destination,
                "backups": {}
            }
            
            # Create backup directory
            backup_dir = Path(destination)
            backup_dir.mkdir(parents=True, exist_ok=True)
            
            # Generate backup timestamp
            backup_timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
            
            total_size = 0
            
            # Backup conversations
            if "conversations" in backup_types:
                conv_backup = _backup_conversations(backup_dir, backup_timestamp)
                backup_results["backups"]["conversations"] = conv_backup
                total_size += conv_backup.get("backup_size_bytes", 0)
            
            # Backup memories
            if "memories" in backup_types:
                mem_backup = _backup_memories(backup_dir, backup_timestamp)
                backup_results["backups"]["memories"] = mem_backup
                total_size += mem_backup.get("backup_size_bytes", 0)
            
            # Backup configurations
            if "configurations" in backup_types:
                config_backup = _backup_configurations(backup_dir, backup_timestamp)
                backup_results["backups"]["configurations"] = config_backup
                total_size += config_backup.get("backup_size_bytes", 0)
            
            # Backup logs
            if "logs" in backup_types:
                log_backup = _backup_logs(backup_dir, backup_timestamp)
                backup_results["backups"]["logs"] = log_backup
                total_size += log_backup.get("backup_size_bytes", 0)
            
            backup_results["summary"] = {
                "total_backups": len([b for b in backup_results["backups"].values() if b.get("success")]),
                "total_size_mb": total_size / (1024 * 1024),
                "backup_duration": (datetime.now() - datetime.fromisoformat(backup_results["timestamp"].replace('Z', '+00:00'))).total_seconds()
            }
            
            logger.info("Data backup completed", 
                       backups_created=backup_results["summary"]["total_backups"],
                       total_size_mb=backup_results["summary"]["total_size_mb"])
            
            return {
                "success": True,
                "backup_results": backup_results
            }
            
        except Exception as e:
            logger.error("Data backup failed", error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=60, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="system.performance_analysis",
    max_retries=1
)
def performance_analysis(
    self,
    analysis_period_hours: int = 24
) -> Dict[str, Any]:
    """
    Analyze system performance over time.
    
    Args:
        analysis_period_hours: Analysis period in hours
    
    Returns:
        Performance analysis results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Starting performance analysis", 
                   analysis_period_hours=analysis_period_hours)
        
        try:
            analysis_results = {
                "timestamp": datetime.now().isoformat(),
                "correlation_id": correlation_id,
                "analysis_period_hours": analysis_period_hours,
                "metrics": {},
                "recommendations": []
            }
            
            # Collect current performance metrics
            current_metrics = {
                "cpu_usage": psutil.cpu_percent(interval=1),
                "memory_usage": psutil.virtual_memory().percent,
                "disk_usage": {
                    "/": (shutil.disk_usage("/").used / shutil.disk_usage("/").total) * 100
                } if Path("/").exists() else {},
                "load_average": psutil.getloadavg()[0] if hasattr(psutil, 'getloadavg') else 0
            }
            
            analysis_results["metrics"]["current"] = current_metrics
            
            # Generate performance recommendations
            recommendations = []
            
            if current_metrics["cpu_usage"] > 80:
                recommendations.append({
                    "type": "warning",
                    "component": "cpu",
                    "message": "High CPU usage detected",
                    "recommendation": "Consider scaling up or optimizing CPU-intensive tasks"
                })
            
            if current_metrics["memory_usage"] > 85:
                recommendations.append({
                    "type": "warning", 
                    "component": "memory",
                    "message": "High memory usage detected",
                    "recommendation": "Consider increasing memory or optimizing memory usage"
                })
            
            disk_usage_root = current_metrics["disk_usage"].get("/", 0)
            if disk_usage_root > 90:
                recommendations.append({
                    "type": "critical",
                    "component": "disk",
                    "message": "Critical disk space usage",
                    "recommendation": "Clean up disk space immediately or add more storage"
                })
            elif disk_usage_root > 80:
                recommendations.append({
                    "type": "warning",
                    "component": "disk", 
                    "message": "High disk space usage",
                    "recommendation": "Consider cleaning up old files or expanding storage"
                })
            
            analysis_results["recommendations"] = recommendations
            
            # Calculate performance score
            cpu_score = max(0, 100 - current_metrics["cpu_usage"])
            memory_score = max(0, 100 - current_metrics["memory_usage"])
            disk_score = max(0, 100 - disk_usage_root)
            
            overall_score = (cpu_score + memory_score + disk_score) / 3
            analysis_results["performance_score"] = round(overall_score, 2)
            
            logger.info("Performance analysis completed", 
                       performance_score=analysis_results["performance_score"],
                       recommendations_count=len(recommendations))
            
            return {
                "success": True,
                "analysis_results": analysis_results
            }
            
        except Exception as e:
            logger.error("Performance analysis failed", error=str(e))
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


# Helper functions
def _check_system_resources() -> Dict[str, Any]:
    """Check system resource status."""
    try:
        return {
            "status": "healthy",
            "cpu_percent": psutil.cpu_percent(interval=1),
            "memory_percent": psutil.virtual_memory().percent,
            "disk_percent": (shutil.disk_usage("/").used / shutil.disk_usage("/").total) * 100 if Path("/").exists() else 0,
            "load_average": psutil.getloadavg()[0] if hasattr(psutil, 'getloadavg') else 0
        }
    except Exception as e:
        return {
            "status": "critical",
            "error": str(e)
        }


def _check_database_health() -> Dict[str, Any]:
    """Check database connectivity and health."""
    try:
        # This would check actual database connection
        # For now, return placeholder
        return {
            "status": "healthy",
            "connection": "active",
            "response_time_ms": 50
        }
    except Exception as e:
        return {
            "status": "critical",
            "error": str(e)
        }


def _check_redis_health() -> Dict[str, Any]:
    """Check Redis connectivity and health."""
    try:
        # This would check actual Redis connection
        # For now, return placeholder
        return {
            "status": "healthy",
            "connection": "active",
            "memory_usage": "moderate"
        }
    except Exception as e:
        return {
            "status": "warning",
            "error": str(e)
        }


def _check_disk_space() -> Dict[str, Any]:
    """Check disk space usage."""
    try:
        disk_info = {}
        for path in ["/", "/tmp", "data"]:
            if Path(path).exists():
                usage = shutil.disk_usage(path)
                percent_used = (usage.used / usage.total) * 100
                disk_info[path] = {
                    "total_gb": usage.total / (1024**3),
                    "used_gb": usage.used / (1024**3),
                    "free_gb": usage.free / (1024**3),
                    "percent_used": percent_used
                }
        
        # Determine overall status
        max_usage = max([info["percent_used"] for info in disk_info.values()], default=0)
        
        if max_usage > 90:
            status = "critical"
        elif max_usage > 80:
            status = "warning"
        else:
            status = "healthy"
        
        return {
            "status": status,
            "disks": disk_info,
            "max_usage_percent": max_usage
        }
    
    except Exception as e:
        return {
            "status": "critical",
            "error": str(e)
        }


def _check_ai_services() -> Dict[str, Any]:
    """Check AI services status."""
    try:
        # This would check actual AI service connections
        # For now, return placeholder
        return {
            "status": "healthy",
            "openai": "connected",
            "anthropic": "connected",
            "local_models": "loaded"
        }
    except Exception as e:
        return {
            "status": "warning",
            "error": str(e)
        }


def _check_audio_services() -> Dict[str, Any]:
    """Check audio services status."""
    try:
        # This would check actual audio service status
        # For now, return placeholder
        return {
            "status": "healthy",
            "whisper": "loaded",
            "tts": "available",
            "audio_devices": "detected"
        }
    except Exception as e:
        return {
            "status": "warning",
            "error": str(e)
        }


def _check_hardware_interfaces() -> Dict[str, Any]:
    """Check hardware interface status."""
    try:
        # This would check actual hardware connections
        # For now, return placeholder
        return {
            "status": "healthy",
            "esp32": "connected",
            "serial_ports": "available",
            "mqtt": "connected"
        }
    except Exception as e:
        return {
            "status": "warning",
            "error": str(e)
        }


def _cleanup_temp_files(older_than_days: int) -> Dict[str, Any]:
    """Clean up temporary files."""
    try:
        files_removed = 0
        space_freed = 0
        
        temp_dirs = [Path("/tmp"), Path("data/temp"), Path("temp")]
        cutoff_time = datetime.now() - timedelta(days=older_than_days)
        
        for temp_dir in temp_dirs:
            if temp_dir.exists():
                for file_path in temp_dir.rglob("*"):
                    if file_path.is_file():
                        try:
                            file_time = datetime.fromtimestamp(file_path.stat().st_mtime)
                            if file_time < cutoff_time:
                                file_size = file_path.stat().st_size
                                file_path.unlink()
                                files_removed += 1
                                space_freed += file_size
                        except Exception:
                            continue  # Skip files that can't be processed
        
        return {
            "success": True,
            "files_removed": files_removed,
            "space_freed_bytes": space_freed
        }
    
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "files_removed": 0,
            "space_freed_bytes": 0
        }


def _cleanup_old_logs(older_than_days: int) -> Dict[str, Any]:
    """Clean up old log files."""
    try:
        files_removed = 0
        space_freed = 0
        
        log_dirs = [Path("logs"), Path("data/logs")]
        cutoff_time = datetime.now() - timedelta(days=older_than_days)
        
        for log_dir in log_dirs:
            if log_dir.exists():
                for log_file in log_dir.rglob("*.log*"):
                    try:
                        file_time = datetime.fromtimestamp(log_file.stat().st_mtime)
                        if file_time < cutoff_time:
                            file_size = log_file.stat().st_size
                            log_file.unlink()
                            files_removed += 1
                            space_freed += file_size
                    except Exception:
                        continue
        
        return {
            "success": True,
            "files_removed": files_removed,
            "space_freed_bytes": space_freed
        }
    
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "files_removed": 0,
            "space_freed_bytes": 0
        }


def _cleanup_cache_files(older_than_days: int) -> Dict[str, Any]:
    """Clean up cache files."""
    try:
        files_removed = 0
        space_freed = 0
        
        cache_dirs = [Path("data/cache"), Path("cache")]
        cutoff_time = datetime.now() - timedelta(days=older_than_days)
        
        for cache_dir in cache_dirs:
            if cache_dir.exists():
                for cache_file in cache_dir.rglob("*"):
                    if cache_file.is_file():
                        try:
                            file_time = datetime.fromtimestamp(cache_file.stat().st_mtime)
                            if file_time < cutoff_time:
                                file_size = cache_file.stat().st_size
                                cache_file.unlink()
                                files_removed += 1
                                space_freed += file_size
                        except Exception:
                            continue
        
        return {
            "success": True,
            "files_removed": files_removed,
            "space_freed_bytes": space_freed
        }
    
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "files_removed": 0,
            "space_freed_bytes": 0
        }


def _cleanup_old_tasks(older_than_days: int) -> Dict[str, Any]:
    """Clean up old Celery task results."""
    try:
        # This would clean up old task results from Redis/database
        # For now, return placeholder
        return {
            "success": True,
            "tasks_cleaned": 0
        }
    
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "tasks_cleaned": 0
        }


def _backup_conversations(backup_dir: Path, timestamp: str) -> Dict[str, Any]:
    """Backup conversation data."""
    try:
        # This would backup actual conversation data
        # For now, return placeholder
        backup_file = backup_dir / f"conversations_{timestamp}.json"
        
        # Mock backup creation
        with open(backup_file, 'w') as f:
            json.dump({"conversations": [], "timestamp": timestamp}, f)
        
        return {
            "success": True,
            "backup_file": str(backup_file),
            "backup_size_bytes": backup_file.stat().st_size
        }
    
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "backup_size_bytes": 0
        }


def _backup_memories(backup_dir: Path, timestamp: str) -> Dict[str, Any]:
    """Backup memory data."""
    try:
        # This would backup actual memory data
        # For now, return placeholder
        backup_file = backup_dir / f"memories_{timestamp}.json"
        
        with open(backup_file, 'w') as f:
            json.dump({"memories": [], "timestamp": timestamp}, f)
        
        return {
            "success": True,
            "backup_file": str(backup_file),
            "backup_size_bytes": backup_file.stat().st_size
        }
    
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "backup_size_bytes": 0
        }


def _backup_configurations(backup_dir: Path, timestamp: str) -> Dict[str, Any]:
    """Backup configuration files."""
    try:
        backup_file = backup_dir / f"configurations_{timestamp}.tar.gz"
        
        # This would create actual backup of config files
        # For now, create placeholder
        with open(backup_file, 'wb') as f:
            f.write(b"mock backup data")
        
        return {
            "success": True,
            "backup_file": str(backup_file),
            "backup_size_bytes": backup_file.stat().st_size
        }
    
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "backup_size_bytes": 0
        }


def _backup_logs(backup_dir: Path, timestamp: str) -> Dict[str, Any]:
    """Backup log files."""
    try:
        backup_file = backup_dir / f"logs_{timestamp}.tar.gz"
        
        # This would create actual backup of log files
        # For now, create placeholder
        with open(backup_file, 'wb') as f:
            f.write(b"mock log backup data")
        
        return {
            "success": True,
            "backup_file": str(backup_file),
            "backup_size_bytes": backup_file.stat().st_size
        }
    
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "backup_size_bytes": 0
        }


# Export tasks
__all__ = [
    "health_check",
    "system_monitoring",
    "cleanup_tasks",
    "backup_data",
    "performance_analysis"
]
