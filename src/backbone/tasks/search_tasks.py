"""
Search Tasks - Celery tasks for web search and information retrieval.
Handles web searches, documentation searches, and code repository searches.
"""

from typing import Dict, Any, List, Optional
from datetime import datetime
import json
import uuid

from celery import current_app
from celery.exceptions import Retry

from src.utils.logging import get_logger, with_correlation_id
from src.config import settings


logger = get_logger(__name__)


@current_app.task(
    bind=True,
    name="search.web_search",
    max_retries=3,
    default_retry_delay=30
)
def web_search(
    self,
    query: str,
    num_results: int = 10,
    safe_search: bool = True,
    region: Optional[str] = None
) -> Dict[str, Any]:
    """
    Perform web search using search engines.
    
    Args:
        query: Search query
        num_results: Number of results to return
        safe_search: Enable safe search filtering
        region: Search region preference
    
    Returns:
        Search results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Performing web search", 
                   query=query[:100] + "..." if len(query) > 100 else query,
                   num_results=num_results)
        
        try:
            # Try multiple search engines
            results = []
            
            # DuckDuckGo search (privacy-focused)
            try:
                results.extend(await _duckduckgo_search(query, num_results // 2))
            except Exception as e:
                logger.warning("DuckDuckGo search failed", error=str(e))
            
            # Bing search (backup)
            try:
                if len(results) < num_results:
                    results.extend(await _bing_search(query, num_results - len(results)))
            except Exception as e:
                logger.warning("Bing search failed", error=str(e))
            
            # Filter and rank results
            filtered_results = _filter_search_results(results, safe_search)
            
            logger.info("Web search completed", 
                       query=query[:50] + "..." if len(query) > 50 else query,
                       results_count=len(filtered_results))
            
            return {
                "success": True,
                "query": query,
                "results": filtered_results[:num_results],
                "total_found": len(filtered_results),
                "sources": ["duckduckgo", "bing"],
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Web search failed", 
                        query=query,
                        error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=30, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "query": query,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="search.search_documentation",
    max_retries=2,
    default_retry_delay=20
)
def search_documentation(
    self,
    query: str,
    doc_types: List[str] = None,
    limit: int = 10
) -> Dict[str, Any]:
    """
    Search through technical documentation.
    
    Args:
        query: Search query
        doc_types: Types of documentation to search
        limit: Maximum results to return
    
    Returns:
        Documentation search results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Searching documentation", 
                   query=query[:50] + "..." if len(query) > 50 else query,
                   doc_types=doc_types or ["all"])
        
        try:
            results = []
            
            # Define documentation sources
            doc_sources = {
                "python": ["https://docs.python.org", "https://pypi.org"],
                "javascript": ["https://developer.mozilla.org", "https://nodejs.org/docs"],
                "react": ["https://react.dev", "https://reactjs.org"],
                "fastapi": ["https://fastapi.tiangolo.com"],
                "docker": ["https://docs.docker.com"],
                "celery": ["https://docs.celeryq.dev"]
            }
            
            # Use specified doc types or search all
            search_types = doc_types or list(doc_sources.keys())
            
            for doc_type in search_types:
                if doc_type in doc_sources:
                    try:
                        doc_results = await _search_documentation_source(
                            query, doc_sources[doc_type], doc_type
                        )
                        results.extend(doc_results)
                    except Exception as e:
                        logger.warning(f"{doc_type} documentation search failed", error=str(e))
            
            # Sort by relevance
            results = _rank_documentation_results(results, query)
            
            logger.info("Documentation search completed", 
                       query=query[:50] + "..." if len(query) > 50 else query,
                       results_count=len(results))
            
            return {
                "success": True,
                "query": query,
                "results": results[:limit],
                "doc_types": search_types,
                "total_found": len(results),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Documentation search failed", 
                        query=query,
                        error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=20, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="search.search_code_repositories",
    max_retries=2,
    default_retry_delay=30
)
def search_code_repositories(
    self,
    query: str,
    language: Optional[str] = None,
    repositories: List[str] = None,
    limit: int = 10
) -> Dict[str, Any]:
    """
    Search through code repositories.
    
    Args:
        query: Code search query
        language: Programming language filter
        repositories: Specific repositories to search
        limit: Maximum results to return
    
    Returns:
        Code search results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Searching code repositories", 
                   query=query[:50] + "..." if len(query) > 50 else query,
                   language=language,
                   repositories=repositories)
        
        try:
            results = []
            
            # GitHub search
            try:
                github_results = await _github_code_search(
                    query, language, repositories, limit // 2
                )
                results.extend(github_results)
            except Exception as e:
                logger.warning("GitHub search failed", error=str(e))
            
            # GitLab search (if configured)
            try:
                if hasattr(settings, 'gitlab_token') and settings.gitlab_token:
                    gitlab_results = await _gitlab_code_search(
                        query, language, limit - len(results)
                    )
                    results.extend(gitlab_results)
            except Exception as e:
                logger.warning("GitLab search failed", error=str(e))
            
            # Filter and rank results
            results = _rank_code_results(results, query, language)
            
            logger.info("Code repository search completed", 
                       query=query[:50] + "..." if len(query) > 50 else query,
                       results_count=len(results))
            
            return {
                "success": True,
                "query": query,
                "results": results[:limit],
                "language": language,
                "repositories": repositories,
                "total_found": len(results),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Code repository search failed", 
                        query=query,
                        error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=30, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


@current_app.task(
    bind=True,
    name="search.intelligent_search",
    max_retries=2
)
def intelligent_search(
    self,
    query: str,
    search_types: List[str] = None,
    context: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Perform intelligent multi-source search with context awareness.
    
    Args:
        query: Search query
        search_types: Types of searches to perform
        context: Search context for better results
    
    Returns:
        Aggregated search results
    """
    correlation_id = str(uuid.uuid4())
    
    with with_correlation_id(correlation_id):
        logger.info("Performing intelligent search", 
                   query=query[:50] + "..." if len(query) > 50 else query,
                   search_types=search_types)
        
        try:
            # Analyze query to determine best search strategy
            search_strategy = await _analyze_search_query(query, context or {})
            
            # Use provided search types or intelligent selection
            if not search_types:
                search_types = search_strategy.get("recommended_types", ["web"])
            
            results = {
                "web": [],
                "documentation": [],
                "code": [],
                "combined": []
            }
            
            # Execute searches in parallel
            search_tasks = []
            
            if "web" in search_types:
                search_tasks.append(
                    web_search.apply_async(args=[query, 5])
                )
            
            if "documentation" in search_types:
                doc_types = search_strategy.get("doc_types", None)
                search_tasks.append(
                    search_documentation.apply_async(args=[query, doc_types, 5])
                )
            
            if "code" in search_types:
                language = search_strategy.get("language", None)
                search_tasks.append(
                    search_code_repositories.apply_async(args=[query, language, None, 5])
                )
            
            # Collect results
            for task in search_tasks:
                try:
                    task_result = task.get(timeout=60)
                    
                    if task_result.get("success"):
                        # Determine result type based on task
                        if "web_search" in str(task):
                            results["web"] = task_result.get("results", [])
                        elif "documentation" in str(task):
                            results["documentation"] = task_result.get("results", [])
                        elif "code" in str(task):
                            results["code"] = task_result.get("results", [])
                
                except Exception as e:
                    logger.warning("Search task failed", error=str(e))
            
            # Combine and rank all results
            all_results = []
            for result_type, type_results in results.items():
                if result_type != "combined":
                    for result in type_results:
                        result["source_type"] = result_type
                        all_results.append(result)
            
            # Intelligent ranking based on query and context
            ranked_results = await _intelligent_ranking(all_results, query, context or {})
            results["combined"] = ranked_results
            
            logger.info("Intelligent search completed", 
                       query=query[:50] + "..." if len(query) > 50 else query,
                       total_results=len(ranked_results))
            
            return {
                "success": True,
                "query": query,
                "results": results,
                "search_strategy": search_strategy,
                "search_types": search_types,
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }
            
        except Exception as e:
            logger.error("Intelligent search failed", 
                        query=query,
                        error=str(e))
            
            if self.request.retries < self.max_retries:
                raise self.retry(countdown=60, exc=e)
            
            return {
                "success": False,
                "error": str(e),
                "correlation_id": correlation_id,
                "timestamp": datetime.now().isoformat()
            }


# Helper functions
async def _duckduckgo_search(query: str, num_results: int) -> List[Dict[str, Any]]:
    """Search using DuckDuckGo."""
    try:
        # This would use duckduckgo-search library
        from duckduckgo_search import DDGS
        
        results = []
        with DDGS() as ddgs:
            for result in ddgs.text(query, max_results=num_results):
                results.append({
                    "title": result.get("title", ""),
                    "url": result.get("href", ""),
                    "snippet": result.get("body", ""),
                    "source": "duckduckgo"
                })
        
        return results
    
    except ImportError:
        logger.warning("duckduckgo-search not available")
        return []
    except Exception as e:
        logger.error("DuckDuckGo search error", error=str(e))
        return []


async def _bing_search(query: str, num_results: int) -> List[Dict[str, Any]]:
    """Search using Bing API."""
    try:
        # This would use Bing Search API
        # Placeholder implementation
        import aiohttp
        
        if not hasattr(settings, 'bing_api_key') or not settings.bing_api_key:
            return []
        
        headers = {"Ocp-Apim-Subscription-Key": settings.bing_api_key}
        params = {"q": query, "count": num_results, "offset": 0}
        
        async with aiohttp.ClientSession() as session:
            async with session.get(
                "https://api.cognitive.microsoft.com/bing/v7.0/search",
                headers=headers,
                params=params
            ) as response:
                if response.status == 200:
                    data = await response.json()
                    results = []
                    
                    for item in data.get("webPages", {}).get("value", []):
                        results.append({
                            "title": item.get("name", ""),
                            "url": item.get("url", ""),
                            "snippet": item.get("snippet", ""),
                            "source": "bing"
                        })
                    
                    return results
        
        return []
    
    except ImportError:
        logger.warning("aiohttp not available for Bing search")
        return []
    except Exception as e:
        logger.error("Bing search error", error=str(e))
        return []


async def _search_documentation_source(
    query: str, 
    sources: List[str], 
    doc_type: str
) -> List[Dict[str, Any]]:
    """Search specific documentation sources."""
    try:
        # This would implement documentation-specific searches
        # For now, return web search results filtered by source
        results = []
        
        for source in sources:
            site_query = f"site:{source} {query}"
            site_results = await _duckduckgo_search(site_query, 3)
            
            for result in site_results:
                result["doc_type"] = doc_type
                result["source"] = f"documentation_{doc_type}"
                results.append(result)
        
        return results
    
    except Exception as e:
        logger.error(f"Documentation search error for {doc_type}", error=str(e))
        return []


async def _github_code_search(
    query: str,
    language: Optional[str],
    repositories: Optional[List[str]],
    limit: int
) -> List[Dict[str, Any]]:
    """Search GitHub code."""
    try:
        # This would use GitHub API
        import aiohttp
        
        search_query = query
        if language:
            search_query += f" language:{language}"
        
        if repositories:
            for repo in repositories:
                search_query += f" repo:{repo}"
        
        headers = {}
        if hasattr(settings, 'github_token') and settings.github_token:
            headers["Authorization"] = f"token {settings.github_token}"
        
        params = {"q": search_query, "per_page": limit}
        
        async with aiohttp.ClientSession() as session:
            async with session.get(
                "https://api.github.com/search/code",
                headers=headers,
                params=params
            ) as response:
                if response.status == 200:
                    data = await response.json()
                    results = []
                    
                    for item in data.get("items", []):
                        results.append({
                            "title": item.get("name", ""),
                            "url": item.get("html_url", ""),
                            "snippet": item.get("text_matches", [{}])[0].get("fragment", "") if item.get("text_matches") else "",
                            "repository": item.get("repository", {}).get("full_name", ""),
                            "path": item.get("path", ""),
                            "language": language,
                            "source": "github"
                        })
                    
                    return results
        
        return []
    
    except Exception as e:
        logger.error("GitHub search error", error=str(e))
        return []


async def _gitlab_code_search(query: str, language: Optional[str], limit: int) -> List[Dict[str, Any]]:
    """Search GitLab code."""
    # Placeholder implementation
    return []


def _filter_search_results(results: List[Dict[str, Any]], safe_search: bool) -> List[Dict[str, Any]]:
    """Filter search results."""
    if not safe_search:
        return results
    
    # Basic filtering - in practice you'd use more sophisticated filtering
    filtered = []
    for result in results:
        # Filter out potentially unsafe content
        title = result.get("title", "").lower()
        snippet = result.get("snippet", "").lower()
        
        if not any(word in title + snippet for word in ["explicit", "adult", "nsfw"]):
            filtered.append(result)
    
    return filtered


def _rank_documentation_results(results: List[Dict[str, Any]], query: str) -> List[Dict[str, Any]]:
    """Rank documentation results by relevance."""
    # Simple scoring based on query terms in title
    query_terms = query.lower().split()
    
    for result in results:
        score = 0
        title = result.get("title", "").lower()
        snippet = result.get("snippet", "").lower()
        
        for term in query_terms:
            if term in title:
                score += 3
            if term in snippet:
                score += 1
        
        result["relevance_score"] = score
    
    # Sort by score
    return sorted(results, key=lambda x: x.get("relevance_score", 0), reverse=True)


def _rank_code_results(results: List[Dict[str, Any]], query: str, language: Optional[str]) -> List[Dict[str, Any]]:
    """Rank code results by relevance."""
    query_terms = query.lower().split()
    
    for result in results:
        score = 0
        title = result.get("title", "").lower()
        snippet = result.get("snippet", "").lower()
        path = result.get("path", "").lower()
        
        # Base scoring
        for term in query_terms:
            if term in title:
                score += 5
            if term in snippet:
                score += 2
            if term in path:
                score += 1
        
        # Language bonus
        if language and result.get("language") == language:
            score += 3
        
        result["relevance_score"] = score
    
    return sorted(results, key=lambda x: x.get("relevance_score", 0), reverse=True)


async def _analyze_search_query(query: str, context: Dict[str, Any]) -> Dict[str, Any]:
    """Analyze query to determine best search strategy."""
    query_lower = query.lower()
    
    # Detect programming-related queries
    programming_keywords = ["function", "class", "method", "api", "library", "framework", "error", "debug", "install"]
    is_programming = any(keyword in query_lower for keyword in programming_keywords)
    
    # Detect language
    language = None
    languages = ["python", "javascript", "java", "cpp", "rust", "go", "typescript", "php"]
    for lang in languages:
        if lang in query_lower:
            language = lang
            break
    
    # Determine recommended search types
    recommended_types = ["web"]
    
    if is_programming:
        recommended_types.extend(["documentation", "code"])
    
    return {
        "is_programming": is_programming,
        "language": language,
        "recommended_types": recommended_types,
        "doc_types": [language] if language else None,
        "confidence": 0.8 if is_programming else 0.5
    }


async def _intelligent_ranking(
    results: List[Dict[str, Any]], 
    query: str, 
    context: Dict[str, Any]
) -> List[Dict[str, Any]]:
    """Intelligent ranking of combined results."""
    query_terms = query.lower().split()
    
    for result in results:
        score = result.get("relevance_score", 0)
        
        # Source type weighting
        source_type = result.get("source_type", "")
        if source_type == "documentation":
            score *= 1.2  # Boost documentation
        elif source_type == "code":
            score *= 1.1  # Slight boost for code
        
        # Recency bonus (if available)
        if "date" in result:
            # Would implement date-based scoring
            pass
        
        result["final_score"] = score
    
    return sorted(results, key=lambda x: x.get("final_score", 0), reverse=True)


# Export tasks
__all__ = [
    "web_search",
    "search_documentation", 
    "search_code_repositories",
    "intelligent_search"
]
