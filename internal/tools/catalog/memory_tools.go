package catalog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/tools"
	"github.com/xpanvictor/xarvis/internal/types"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

// MemorySearchToolBuilder builds a tool to search user's memories
type MemorySearchToolBuilder struct{}

func (m *MemorySearchToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("search_memories", "1.0.0", "Search user's memories by content and type").
		AddStringParameter("query", "Search query to match against memory content", true).
		AddStringParameter("memory_type", "Optional memory type filter: 'episodic' or 'semantic'", false).
		AddNumberParameter("limit", "Maximum number of memories to return (default: 20)", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Extract user ID from injected context (secure - cannot be manipulated)
			userIDStr, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available - this is a system error")
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return nil, fmt.Errorf("invalid user_id format in context: %w", err)
			}

			query, ok := args["query"].(string)
			if !ok {
				return nil, fmt.Errorf("query parameter is required and must be a string")
			}

			// Set default limit
			limit := 20
			if l, exists := args["limit"]; exists {
				if limitFloat, ok := l.(float64); ok {
					limit = int(limitFloat)
				}
			}

			// Parse memory type if provided
			var memoryType *types.MemoryType
			if mt, exists := args["memory_type"]; exists {
				if mtStr, ok := mt.(string); ok {
					mtStr = strings.ToLower(strings.TrimSpace(mtStr))
					switch mtStr {
					case "episodic":
						episodicType := types.EPISODIC
						memoryType = &episodicType
					case "semantic":
						semanticType := types.SEMANTIC
						memoryType = &semanticType
					default:
						return nil, fmt.Errorf("invalid memory_type '%s', must be 'episodic' or 'semantic'", mtStr)
					}
				}
			}

			// Search memories using the conversation service
			memories, err := deps.ConversationService.SearchMemories(ctx, userID, query, memoryType, limit)
			if err != nil {
				return nil, fmt.Errorf("failed to search memories: %w", err)
			}

			// Return formatted response
			return map[string]any{
				"memories": memories,
				"count":    len(memories),
				"query":    query,
			}, nil
		}).
		AddTags("memory", "search", "recall").
		Build()
}

// MemoryCreateToolBuilder builds a tool to create new memories
type MemoryCreateToolBuilder struct{}

func (m *MemoryCreateToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("create_memory", "1.0.0", "Create a new memory for the user").
		AddStringParameter("content", "The memory content", true).
		AddStringParameter("memory_type", "Memory type: 'episodic' or 'semantic'", true).
		AddNumberParameter("saliency_score", "Optional saliency score (1-255, default: 1)", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Extract user ID from injected context (secure - cannot be manipulated)
			userIDStr, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available - this is a system error")
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return nil, fmt.Errorf("invalid user_id format in context: %w", err)
			}

			content, ok := args["content"].(string)
			if !ok {
				return nil, fmt.Errorf("content parameter is required and must be a string")
			}

			memoryTypeStr, ok := args["memory_type"].(string)
			if !ok {
				return nil, fmt.Errorf("memory_type parameter is required and must be a string")
			}

			// Parse memory type
			memoryTypeStr = strings.ToLower(strings.TrimSpace(memoryTypeStr))
			var memoryType types.MemoryType
			switch memoryTypeStr {
			case "episodic":
				memoryType = types.EPISODIC
			case "semantic":
				memoryType = types.SEMANTIC
			default:
				return nil, fmt.Errorf("invalid memory_type '%s', must be 'episodic' or 'semantic'", memoryTypeStr)
			}

			// Parse saliency score if provided
			saliencyScore := uint8(1) // default
			if ss, exists := args["saliency_score"]; exists {
				if ssFloat, ok := ss.(float64); ok {
					if ssFloat < 1 || ssFloat > 255 {
						return nil, fmt.Errorf("saliency_score must be between 1 and 255")
					}
					saliencyScore = uint8(ssFloat)
				}
			}

			// Create memory object
			memory := types.Memory{
				Type:          memoryType,
				Content:       content,
				SaliencyScore: saliencyScore,
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}

			// Create memory using the conversation service
			createdMemory, err := deps.ConversationService.CreateMemory(ctx, userID, memory)
			if err != nil {
				return nil, fmt.Errorf("failed to create memory: %w", err)
			}

			// Return formatted response
			return map[string]any{
				"success":    true,
				"memory":     createdMemory,
				"message":    "Memory created successfully",
				"memory_id":  createdMemory.ID,
				"created_at": createdMemory.CreatedAt,
			}, nil
		}).
		AddTags("memory", "create", "store").
		Build()
}

// MemoryListToolBuilder builds a tool to list user's memories with optional filters
type MemoryListToolBuilder struct{}

func (m *MemoryListToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("list_memories", "1.0.0", "List user's memories with optional filtering").
		AddStringParameter("memory_type", "Optional memory type filter: 'episodic' or 'semantic'", false).
		AddNumberParameter("limit", "Maximum number of memories to return (default: 50)", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Debug: Check if deps is nil
			if deps == nil {
				return nil, fmt.Errorf("tool dependencies are nil - this is a system configuration error")
			}

			// Debug: Check if ConversationService is nil
			if deps.ConversationService == nil {
				return nil, fmt.Errorf("ConversationService is nil in tool dependencies - check app setup")
			}

			// Extract user ID from injected context (secure - cannot be manipulated)
			userIDStr, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available - this is a system error")
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return nil, fmt.Errorf("invalid user_id format in context: %w", err)
			}

			// Set default limit
			limit := 50
			if l, exists := args["limit"]; exists {
				if limitFloat, ok := l.(float64); ok {
					limit = int(limitFloat)
				}
			}

			// Parse memory type if provided
			var memoryType *types.MemoryType
			if mt, exists := args["memory_type"]; exists {
				if mtStr, ok := mt.(string); ok {
					mtStr = strings.ToLower(strings.TrimSpace(mtStr))
					switch mtStr {
					case "episodic":
						episodicType := types.EPISODIC
						memoryType = &episodicType
					case "semantic":
						semanticType := types.SEMANTIC
						memoryType = &semanticType
					default:
						return nil, fmt.Errorf("invalid memory_type '%s', must be 'episodic' or 'semantic'", mtStr)
					}
				}
			}

			// List memories using search with empty query (returns all)
			memories, err := deps.ConversationService.SearchMemories(ctx, userID, "", memoryType, limit)
			if err != nil {
				return nil, fmt.Errorf("failed to list memories: %w", err)
			}

			// Return formatted response
			response := map[string]any{
				"memories": memories,
				"count":    len(memories),
				"total":    len(memories), // For now, same as count since no pagination
			}

			// Add filtering info if applied
			if memoryType != nil {
				response["filtered_by_type"] = string(*memoryType)
			}

			return response, nil
		}).
		AddTags("memory", "list", "filter").
		Build()
}
