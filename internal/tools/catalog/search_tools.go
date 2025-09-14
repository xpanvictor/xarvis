package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/xpanvictor/xarvis/internal/tools"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

// TavilySearchRequest represents the request structure for Tavily API
type TavilySearchRequest struct {
	APIKey         string   `json:"api_key"`
	Query          string   `json:"query"`
	SearchDepth    string   `json:"search_depth,omitempty"` // "basic" or "advanced"
	IncludeImages  bool     `json:"include_images,omitempty"`
	IncludeAnswer  bool     `json:"include_answer,omitempty"`
	IncludeDomains []string `json:"include_domains,omitempty"`
	ExcludeDomains []string `json:"exclude_domains,omitempty"`
	MaxResults     int      `json:"max_results,omitempty"`
}

// TavilySearchResult represents a single search result from Tavily API
type TavilySearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// TavilySearchResponse represents the response structure from Tavily API
type TavilySearchResponse struct {
	Query   string               `json:"query"`
	Answer  string               `json:"answer,omitempty"`
	Results []TavilySearchResult `json:"results"`
	Images  []string             `json:"images,omitempty"`
}

// WebSearchToolBuilder builds a tool to search the web using Tavily API
type WebSearchToolBuilder struct{}

func (w *WebSearchToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("web_search", "1.0.0", "Search the web for current information using Tavily API").
		AddStringParameter("query", "Search query to find information on the web", true).
		AddStringParameter("search_depth", "Search depth: 'basic' for quick results or 'advanced' for comprehensive search", false).
		AddBooleanParameter("include_answer", "Whether to include AI-generated answer summary", false).
		AddBooleanParameter("include_images", "Whether to include related images in results", false).
		AddNumberParameter("max_results", "Maximum number of search results to return (1-20)", false).
		AddArrayParameter("include_domains", "Specific domains to search within", false).
		AddArrayParameter("exclude_domains", "Domains to exclude from search", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			query, ok := args["query"].(string)
			if !ok {
				return nil, fmt.Errorf("query parameter is required and must be a string")
			}

			if deps.TavilyAPIKey == "" {
				return nil, fmt.Errorf("tavily API key is not configured")
			}

			// Build search request
			searchReq := TavilySearchRequest{
				APIKey:        deps.TavilyAPIKey,
				Query:         query,
				SearchDepth:   "basic", // default
				IncludeAnswer: true,    // default
				MaxResults:    5,       // default
			}

			// Set search depth
			if depth, exists := args["search_depth"]; exists {
				if depthStr, ok := depth.(string); ok && (depthStr == "basic" || depthStr == "advanced") {
					searchReq.SearchDepth = depthStr
				}
			}

			// Set include answer
			if includeAnswer, exists := args["include_answer"]; exists {
				if includeAnswerBool, ok := includeAnswer.(bool); ok {
					searchReq.IncludeAnswer = includeAnswerBool
				}
			}

			// Set include images
			if includeImages, exists := args["include_images"]; exists {
				if includeImagesBool, ok := includeImages.(bool); ok {
					searchReq.IncludeImages = includeImagesBool
				}
			}

			// Set max results
			if maxResults, exists := args["max_results"]; exists {
				if maxResultsFloat, ok := maxResults.(float64); ok {
					maxRes := int(maxResultsFloat)
					if maxRes >= 1 && maxRes <= 20 {
						searchReq.MaxResults = maxRes
					}
				}
			}

			// Set include domains
			if includeDomains, exists := args["include_domains"]; exists {
				if domainArray, ok := includeDomains.([]interface{}); ok {
					var domains []string
					for _, domain := range domainArray {
						if domainStr, ok := domain.(string); ok {
							domains = append(domains, domainStr)
						}
					}
					searchReq.IncludeDomains = domains
				}
			}

			// Set exclude domains
			if excludeDomains, exists := args["exclude_domains"]; exists {
				if domainArray, ok := excludeDomains.([]interface{}); ok {
					var domains []string
					for _, domain := range domainArray {
						if domainStr, ok := domain.(string); ok {
							domains = append(domains, domainStr)
						}
					}
					searchReq.ExcludeDomains = domains
				}
			}

			// Make API request
			searchResp, err := makeTavilyRequest(ctx, searchReq)
			if err != nil {
				deps.Logger.Error("Tavily search failed: %v", err)
				return nil, fmt.Errorf("web search failed: %w", err)
			}

			// Format response
			result := map[string]any{
				"query":        searchResp.Query,
				"results":      searchResp.Results,
				"result_count": len(searchResp.Results),
				"success":      true,
			}

			if searchResp.Answer != "" {
				result["answer"] = searchResp.Answer
			}

			if len(searchResp.Images) > 0 {
				result["images"] = searchResp.Images
			}

			deps.Logger.Info("Web search completed successfully: query='%s', results=%d", query, len(searchResp.Results))

			return result, nil
		}).
		AddTags("web", "search", "internet", "information").
		Build()
}

// makeTavilyRequest performs the actual HTTP request to Tavily API
func makeTavilyRequest(ctx context.Context, req TavilySearchRequest) (*TavilySearchResponse, error) {
	// Tavily API endpoint
	const tavilyURL = "https://api.tavily.com/search"

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", tavilyURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var searchResp TavilySearchResponse
	if err := json.Unmarshal(respBody, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &searchResp, nil
}
