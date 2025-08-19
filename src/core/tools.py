"""
Tool Registry - Dynamic tool management and execution system.
Provides a framework for self-injection and tool discovery.
"""

from typing import Dict, Any, List, Optional, Callable, Union
from dataclasses import dataclass
from enum import Enum
from datetime import datetime
import inspect
import json
import asyncio
from pathlib import Path
import importlib
import sys

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


class ToolType(Enum):
    """Types of tools available."""
    SYSTEM = "system"
    WEB = "web"
    CALCULATION = "calculation"
    FILE = "file"
    DATABASE = "database"
    API = "api"
    AUTOMATION = "automation"
    ANALYSIS = "analysis"
    GENERATION = "generation"
    COMMUNICATION = "communication"


class ToolStatus(Enum):
    """Tool execution status."""
    AVAILABLE = "available"
    BUSY = "busy"
    ERROR = "error"
    DISABLED = "disabled"
    UNKNOWN = "unknown"


@dataclass
class ToolParameter:
    """Represents a tool parameter."""
    name: str
    type: str
    description: str
    required: bool = True
    default: Any = None


@dataclass
class ToolDefinition:
    """Represents a tool definition."""
    name: str
    description: str
    tool_type: ToolType
    parameters: List[ToolParameter]
    return_type: str
    examples: List[Dict[str, Any]]
    version: str
    author: str
    created_at: datetime
    updated_at: datetime
    usage_count: int
    success_rate: float
    average_execution_time: float


@dataclass
class ToolResult:
    """Represents the result of tool execution."""
    tool_name: str
    success: bool
    result: Any
    error: Optional[str]
    execution_time: float
    timestamp: datetime
    metadata: Dict[str, Any]


class Tool:
    """Base class for all tools."""
    
    def __init__(self, name: str, description: str, tool_type: ToolType):
        self.name = name
        self.description = description
        self.tool_type = tool_type
        self.status = ToolStatus.AVAILABLE
        self.usage_count = 0
        self.success_count = 0
        self.total_execution_time = 0.0
        self.created_at = datetime.now()
        self.updated_at = datetime.now()
    
    async def execute(self, **kwargs) -> ToolResult:
        """Execute the tool with given parameters."""
        start_time = datetime.now()
        
        try:
            self.status = ToolStatus.BUSY
            self.usage_count += 1
            
            # Validate parameters
            await self._validate_parameters(kwargs)
            
            # Execute the actual tool logic
            result = await self._execute_impl(**kwargs)
            
            # Mark as successful
            self.success_count += 1
            execution_time = (datetime.now() - start_time).total_seconds()
            self.total_execution_time += execution_time
            
            self.status = ToolStatus.AVAILABLE
            
            return ToolResult(
                tool_name=self.name,
                success=True,
                result=result,
                error=None,
                execution_time=execution_time,
                timestamp=datetime.now(),
                metadata={"parameters": kwargs}
            )
            
        except Exception as e:
            execution_time = (datetime.now() - start_time).total_seconds()
            self.total_execution_time += execution_time
            self.status = ToolStatus.ERROR
            
            logger.error("Tool execution failed", 
                        tool=self.name, 
                        error=str(e),
                        parameters=kwargs)
            
            return ToolResult(
                tool_name=self.name,
                success=False,
                result=None,
                error=str(e),
                execution_time=execution_time,
                timestamp=datetime.now(),
                metadata={"parameters": kwargs}
            )
    
    async def _execute_impl(self, **kwargs) -> Any:
        """Implementation-specific execution logic. Override in subclasses."""
        raise NotImplementedError("Tools must implement _execute_impl method")
    
    async def _validate_parameters(self, kwargs: Dict[str, Any]) -> None:
        """Validate input parameters. Override in subclasses for custom validation."""
        pass
    
    def get_definition(self) -> ToolDefinition:
        """Get tool definition."""
        return ToolDefinition(
            name=self.name,
            description=self.description,
            tool_type=self.tool_type,
            parameters=self._get_parameters(),
            return_type=self._get_return_type(),
            examples=self._get_examples(),
            version="1.0.0",
            author="Xarvis System",
            created_at=self.created_at,
            updated_at=self.updated_at,
            usage_count=self.usage_count,
            success_rate=self.success_count / max(1, self.usage_count),
            average_execution_time=self.total_execution_time / max(1, self.usage_count)
        )
    
    def _get_parameters(self) -> List[ToolParameter]:
        """Get tool parameters. Override in subclasses."""
        return []
    
    def _get_return_type(self) -> str:
        """Get return type description. Override in subclasses."""
        return "Any"
    
    def _get_examples(self) -> List[Dict[str, Any]]:
        """Get usage examples. Override in subclasses."""
        return []


class ToolRegistry:
    """
    Dynamic tool registry and management system.
    Supports tool discovery, registration, execution, and self-injection.
    """
    
    def __init__(self):
        self.tools: Dict[str, Tool] = {}
        self.tool_categories: Dict[ToolType, List[str]] = {}
        self.execution_history: List[ToolResult] = []
        self.auto_discovery_enabled = True
        self.tools_directory = Path("src/tools")
        
    async def initialize(self) -> None:
        """Initialize the tool registry."""
        logger.info("Initializing Tool Registry")
        
        try:
            # Create tools directory if it doesn't exist
            self.tools_directory.mkdir(parents=True, exist_ok=True)
            
            # Register built-in tools
            await self._register_builtin_tools()
            
            # Auto-discover tools if enabled
            if self.auto_discovery_enabled:
                await self._auto_discover_tools()
            
            # Load tool definitions from disk
            await self._load_tool_definitions()
            
            logger.info("Tool Registry initialized", 
                       tools_count=len(self.tools),
                       categories=len(self.tool_categories))
            
        except Exception as e:
            logger.error("Failed to initialize Tool Registry", error=str(e))
            raise
    
    async def register_tool(self, tool: Tool) -> bool:
        """
        Register a new tool.
        
        Args:
            tool: Tool instance to register
        
        Returns:
            True if successful
        """
        try:
            if tool.name in self.tools:
                logger.warning("Tool already registered, updating", tool_name=tool.name)
            
            self.tools[tool.name] = tool
            
            # Add to category
            if tool.tool_type not in self.tool_categories:
                self.tool_categories[tool.tool_type] = []
            
            if tool.name not in self.tool_categories[tool.tool_type]:
                self.tool_categories[tool.tool_type].append(tool.name)
            
            # Save tool definition
            await self._save_tool_definition(tool)
            
            logger.info("Tool registered successfully", tool_name=tool.name, tool_type=tool.tool_type.value)
            return True
            
        except Exception as e:
            logger.error("Failed to register tool", tool_name=tool.name, error=str(e))
            return False
    
    async def unregister_tool(self, tool_name: str) -> bool:
        """
        Unregister a tool.
        
        Args:
            tool_name: Name of tool to unregister
        
        Returns:
            True if successful
        """
        try:
            if tool_name not in self.tools:
                logger.warning("Tool not found for unregistration", tool_name=tool_name)
                return False
            
            tool = self.tools[tool_name]
            
            # Remove from category
            if tool.tool_type in self.tool_categories:
                if tool_name in self.tool_categories[tool.tool_type]:
                    self.tool_categories[tool.tool_type].remove(tool_name)
            
            # Remove from tools
            del self.tools[tool_name]
            
            # Remove saved definition
            await self._delete_tool_definition(tool_name)
            
            logger.info("Tool unregistered successfully", tool_name=tool_name)
            return True
            
        except Exception as e:
            logger.error("Failed to unregister tool", tool_name=tool_name, error=str(e))
            return False
    
    async def execute_tool(
        self,
        tool_name: str,
        parameters: Dict[str, Any],
        correlation_id: Optional[str] = None
    ) -> ToolResult:
        """
        Execute a tool with given parameters.
        
        Args:
            tool_name: Name of tool to execute
            parameters: Parameters for tool execution
            correlation_id: Optional correlation ID
        
        Returns:
            Tool execution result
        """
        correlation_id = correlation_id or f"tool_exec_{datetime.now().timestamp()}"
        
        with with_correlation_id(correlation_id):
            logger.info("Executing tool", tool_name=tool_name, parameters=parameters)
            
            if tool_name not in self.tools:
                error_msg = f"Tool '{tool_name}' not found"
                logger.error(error_msg)
                
                return ToolResult(
                    tool_name=tool_name,
                    success=False,
                    result=None,
                    error=error_msg,
                    execution_time=0.0,
                    timestamp=datetime.now(),
                    metadata={"parameters": parameters}
                )
            
            try:
                tool = self.tools[tool_name]
                result = await tool.execute(**parameters)
                
                # Store execution history
                self.execution_history.append(result)
                
                # Keep only last 1000 executions
                if len(self.execution_history) > 1000:
                    self.execution_history = self.execution_history[-1000:]
                
                logger.info("Tool executed successfully", 
                           tool_name=tool_name,
                           success=result.success,
                           execution_time=result.execution_time)
                
                return result
                
            except Exception as e:
                error_msg = f"Tool execution failed: {str(e)}"
                logger.error(error_msg, tool_name=tool_name)
                
                return ToolResult(
                    tool_name=tool_name,
                    success=False,
                    result=None,
                    error=error_msg,
                    execution_time=0.0,
                    timestamp=datetime.now(),
                    metadata={"parameters": parameters}
                )
    
    async def list_tools(
        self,
        tool_type: Optional[ToolType] = None,
        status: Optional[ToolStatus] = None
    ) -> List[ToolDefinition]:
        """
        List available tools.
        
        Args:
            tool_type: Filter by tool type
            status: Filter by status
        
        Returns:
            List of tool definitions
        """
        tools = []
        
        for tool in self.tools.values():
            # Filter by type
            if tool_type and tool.tool_type != tool_type:
                continue
            
            # Filter by status
            if status and tool.status != status:
                continue
            
            tools.append(tool.get_definition())
        
        return tools
    
    async def search_tools(
        self,
        query: str,
        tool_type: Optional[ToolType] = None
    ) -> List[ToolDefinition]:
        """
        Search tools by description or name.
        
        Args:
            query: Search query
            tool_type: Optional tool type filter
        
        Returns:
            List of matching tool definitions
        """
        query_lower = query.lower()
        matching_tools = []
        
        for tool in self.tools.values():
            # Filter by type
            if tool_type and tool.tool_type != tool_type:
                continue
            
            # Search in name and description
            if (query_lower in tool.name.lower() or 
                query_lower in tool.description.lower()):
                matching_tools.append(tool.get_definition())
        
        # Sort by relevance (simple scoring)
        def relevance_score(tool_def):
            score = 0
            if query_lower in tool_def.name.lower():
                score += 2
            if query_lower in tool_def.description.lower():
                score += 1
            return score
        
        matching_tools.sort(key=relevance_score, reverse=True)
        return matching_tools
    
    async def inject_tool_from_code(
        self,
        code: str,
        tool_name: str,
        description: str,
        tool_type: ToolType,
        author: str = "AI Generated"
    ) -> bool:
        """
        Inject a new tool from generated code.
        
        Args:
            code: Python code for the tool
            tool_name: Name for the new tool
            description: Tool description
            tool_type: Type of tool
            author: Tool author
        
        Returns:
            True if successful
        """
        try:
            logger.info("Injecting new tool from code", tool_name=tool_name)
            
            # Create tool file
            tool_file = self.tools_directory / f"{tool_name}_generated.py"
            
            # Wrap code in a proper tool class
            wrapped_code = self._wrap_tool_code(code, tool_name, description, tool_type)
            
            # Write to file
            with open(tool_file, 'w') as f:
                f.write(wrapped_code)
            
            # Import and register the tool
            spec = importlib.util.spec_from_file_location(f"{tool_name}_generated", tool_file)
            module = importlib.util.module_from_spec(spec)
            spec.loader.exec_module(module)
            
            # Get the tool class and instantiate
            tool_class = getattr(module, f"{tool_name.title()}Tool")
            tool_instance = tool_class()
            
            # Register the tool
            success = await self.register_tool(tool_instance)
            
            if success:
                logger.info("Tool injected successfully", tool_name=tool_name)
            else:
                logger.error("Failed to register injected tool", tool_name=tool_name)
            
            return success
            
        except Exception as e:
            logger.error("Tool injection failed", tool_name=tool_name, error=str(e))
            return False
    
    async def update_tool(
        self,
        tool_name: str,
        updates: Dict[str, Any]
    ) -> bool:
        """
        Update an existing tool.
        
        Args:
            tool_name: Name of tool to update
            updates: Updates to apply
        
        Returns:
            True if successful
        """
        try:
            if tool_name not in self.tools:
                logger.error("Tool not found for update", tool_name=tool_name)
                return False
            
            tool = self.tools[tool_name]
            
            # Apply updates
            for key, value in updates.items():
                if hasattr(tool, key):
                    setattr(tool, key, value)
            
            tool.updated_at = datetime.now()
            
            # Save updated definition
            await self._save_tool_definition(tool)
            
            logger.info("Tool updated successfully", tool_name=tool_name, updates=list(updates.keys()))
            return True
            
        except Exception as e:
            logger.error("Tool update failed", tool_name=tool_name, error=str(e))
            return False
    
    def get_execution_stats(self) -> Dict[str, Any]:
        """Get tool execution statistics."""
        if not self.execution_history:
            return {"total_executions": 0}
        
        total_executions = len(self.execution_history)
        successful_executions = sum(1 for result in self.execution_history if result.success)
        total_time = sum(result.execution_time for result in self.execution_history)
        
        # Tool usage stats
        tool_usage = {}
        for result in self.execution_history:
            if result.tool_name not in tool_usage:
                tool_usage[result.tool_name] = {"count": 0, "success": 0}
            tool_usage[result.tool_name]["count"] += 1
            if result.success:
                tool_usage[result.tool_name]["success"] += 1
        
        return {
            "total_executions": total_executions,
            "successful_executions": successful_executions,
            "success_rate": successful_executions / total_executions if total_executions > 0 else 0,
            "average_execution_time": total_time / total_executions if total_executions > 0 else 0,
            "tool_usage": tool_usage,
            "most_used_tools": sorted(tool_usage.items(), key=lambda x: x[1]["count"], reverse=True)[:10]
        }
    
    async def _register_builtin_tools(self) -> None:
        """Register built-in system tools."""
        # System status tool
        status_tool = SystemStatusTool()
        await self.register_tool(status_tool)
        
        # Calculator tool
        calc_tool = CalculatorTool()
        await self.register_tool(calc_tool)
        
        # Web search tool
        search_tool = WebSearchTool()
        await self.register_tool(search_tool)
    
    async def _auto_discover_tools(self) -> None:
        """Auto-discover tools in the tools directory."""
        try:
            if not self.tools_directory.exists():
                return
            
            for tool_file in self.tools_directory.glob("*.py"):
                if tool_file.name.startswith("__"):
                    continue
                
                try:
                    # Import the module
                    spec = importlib.util.spec_from_file_location(tool_file.stem, tool_file)
                    module = importlib.util.module_from_spec(spec)
                    spec.loader.exec_module(module)
                    
                    # Look for Tool classes
                    for name in dir(module):
                        obj = getattr(module, name)
                        if (inspect.isclass(obj) and 
                            issubclass(obj, Tool) and 
                            obj != Tool):
                            # Instantiate and register
                            tool_instance = obj()
                            await self.register_tool(tool_instance)
                
                except Exception as e:
                    logger.error("Failed to discover tool", file=str(tool_file), error=str(e))
        
        except Exception as e:
            logger.error("Tool auto-discovery failed", error=str(e))
    
    async def _save_tool_definition(self, tool: Tool) -> None:
        """Save tool definition to disk."""
        try:
            definition_file = self.tools_directory / f"{tool.name}_definition.json"
            definition = tool.get_definition()
            
            # Convert to JSON-serializable format
            definition_dict = {
                "name": definition.name,
                "description": definition.description,
                "tool_type": definition.tool_type.value,
                "version": definition.version,
                "author": definition.author,
                "created_at": definition.created_at.isoformat(),
                "updated_at": definition.updated_at.isoformat(),
                "usage_count": definition.usage_count,
                "success_rate": definition.success_rate,
                "average_execution_time": definition.average_execution_time
            }
            
            with open(definition_file, 'w') as f:
                json.dump(definition_dict, f, indent=2)
        
        except Exception as e:
            logger.error("Failed to save tool definition", tool_name=tool.name, error=str(e))
    
    async def _load_tool_definitions(self) -> None:
        """Load tool definitions from disk."""
        try:
            for definition_file in self.tools_directory.glob("*_definition.json"):
                try:
                    with open(definition_file, 'r') as f:
                        definition_dict = json.load(f)
                    
                    # Update tool stats if tool is registered
                    tool_name = definition_dict["name"]
                    if tool_name in self.tools:
                        tool = self.tools[tool_name]
                        tool.usage_count = definition_dict.get("usage_count", 0)
                        tool.success_count = int(tool.usage_count * definition_dict.get("success_rate", 0))
                        tool.total_execution_time = tool.usage_count * definition_dict.get("average_execution_time", 0)
                
                except Exception as e:
                    logger.error("Failed to load tool definition", file=str(definition_file), error=str(e))
        
        except Exception as e:
            logger.error("Failed to load tool definitions", error=str(e))
    
    async def _delete_tool_definition(self, tool_name: str) -> None:
        """Delete tool definition from disk."""
        try:
            definition_file = self.tools_directory / f"{tool_name}_definition.json"
            if definition_file.exists():
                definition_file.unlink()
        except Exception as e:
            logger.error("Failed to delete tool definition", tool_name=tool_name, error=str(e))
    
    def _wrap_tool_code(
        self,
        code: str,
        tool_name: str,
        description: str,
        tool_type: ToolType
    ) -> str:
        """Wrap user code in a proper tool class."""
        class_name = f"{tool_name.title()}Tool"
        
        return f'''"""
Auto-generated tool: {tool_name}
"""

import asyncio
from typing import Any, Dict, List
from datetime import datetime
from src.core.tools import Tool, ToolType, ToolParameter

class {class_name}(Tool):
    """Auto-generated tool: {description}"""
    
    def __init__(self):
        super().__init__(
            name="{tool_name}",
            description="{description}",
            tool_type=ToolType.{tool_type.name}
        )
    
    async def _execute_impl(self, **kwargs) -> Any:
        """Execute the tool logic."""
        # User code starts here
{code}
        # User code ends here
    
    def _get_parameters(self) -> List[ToolParameter]:
        """Get tool parameters."""
        # This would be auto-generated based on code analysis
        return []
    
    def _get_return_type(self) -> str:
        """Get return type."""
        return "Any"
'''


# Built-in tools
class SystemStatusTool(Tool):
    """Tool for checking system status."""
    
    def __init__(self):
        super().__init__(
            name="system_status",
            description="Check system health and status",
            tool_type=ToolType.SYSTEM
        )
    
    async def _execute_impl(self, **kwargs) -> Dict[str, Any]:
        """Get system status."""
        import psutil
        import platform
        
        return {
            "cpu_percent": psutil.cpu_percent(interval=1),
            "memory_percent": psutil.virtual_memory().percent,
            "disk_percent": psutil.disk_usage('/').percent,
            "platform": platform.system(),
            "python_version": platform.python_version(),
            "timestamp": datetime.now().isoformat()
        }


class CalculatorTool(Tool):
    """Tool for mathematical calculations."""
    
    def __init__(self):
        super().__init__(
            name="calculator",
            description="Perform mathematical calculations",
            tool_type=ToolType.CALCULATION
        )
    
    async def _execute_impl(self, expression: str, **kwargs) -> Union[float, int, str]:
        """Evaluate mathematical expression."""
        try:
            # Simple expression evaluation (be careful in production)
            allowed_names = {
                k: v for k, v in __builtins__.items()
                if k in ['abs', 'round', 'min', 'max', 'sum', 'pow']
            }
            allowed_names.update({
                'pi': 3.14159265359,
                'e': 2.71828182846
            })
            
            result = eval(expression, {"__builtins__": {}}, allowed_names)
            return result
        
        except Exception as e:
            return f"Calculation error: {str(e)}"


class WebSearchTool(Tool):
    """Tool for web searching."""
    
    def __init__(self):
        super().__init__(
            name="web_search",
            description="Search the web for information",
            tool_type=ToolType.WEB
        )
    
    async def _execute_impl(self, query: str, max_results: int = 5, **kwargs) -> List[Dict[str, Any]]:
        """Perform web search."""
        try:
            from duckduckgo_search import DDGS
            
            results = []
            with DDGS() as ddgs:
                for result in ddgs.text(query, max_results=max_results):
                    results.append({
                        "title": result.get("title", ""),
                        "url": result.get("href", ""),
                        "snippet": result.get("body", "")
                    })
            
            return results
        
        except Exception as e:
            return [{"error": f"Search failed: {str(e)}"}]
