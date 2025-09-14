package catalog

import (
	"context"
	"fmt"
	"strings"

	"github.com/xpanvictor/xarvis/internal/domains/note"
	"github.com/xpanvictor/xarvis/internal/tools"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

// Helper function for case-insensitive string matching
func containsIgnoreCase(text, substr string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substr))
}

// NoteListToolBuilder builds a tool to fetch user's notes
type NoteListToolBuilder struct{}

func (n *NoteListToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("fetch_notes", "1.0.0", "Fetch notes for the current user with optional project filtering").
		AddStringParameter("user_id", "The user ID to fetch notes for", true).
		AddStringParameter("project_id", "Optional project ID to filter notes", false).
		AddNumberParameter("limit", "Maximum number of notes to return", false).
		AddNumberParameter("offset", "Number of notes to skip for pagination", false).
		AddStringParameter("search", "Search term to filter notes by content", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			userID, ok := args["user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user_id parameter is required and must be a string")
			}

			// Set defaults
			limit := 50
			offset := 0

			if l, exists := args["limit"]; exists {
				if limitFloat, ok := l.(float64); ok {
					limit = int(limitFloat)
				}
			}

			if o, exists := args["offset"]; exists {
				if offsetFloat, ok := o.(float64); ok {
					offset = int(offsetFloat)
				}
			}

			// Check if filtering by project
			if projectID, exists := args["project_id"]; exists {
				if projIDStr, ok := projectID.(string); ok {
					notes, total, err := deps.NoteService.GetProjectNotes(ctx, userID, projIDStr, offset, limit)
					if err != nil {
						return nil, fmt.Errorf("failed to fetch project notes: %w", err)
					}

					return map[string]any{
						"notes":       notes,
						"total_count": total,
						"returned":    len(notes),
						"offset":      offset,
						"limit":       limit,
						"project_id":  projIDStr,
					}, nil
				}
			}

			// Create list request for user notes
			listReq := note.ListNotesRequest{
				Offset: offset,
				Limit:  limit,
			}

			// Add search if provided
			if search, exists := args["search"]; exists {
				if searchStr, ok := search.(string); ok {
					listReq.Search = searchStr
				}
			}

			notes, total, err := deps.NoteService.ListUserNotes(ctx, userID, listReq)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch notes: %w", err)
			}

			return map[string]any{
				"notes":       notes,
				"total_count": total,
				"returned":    len(notes),
				"offset":      offset,
				"limit":       limit,
			}, nil
		}).
		AddTags("note", "list", "user").
		Build()
}

// NoteCreateToolBuilder builds a tool to create new notes
type NoteCreateToolBuilder struct{}

func (n *NoteCreateToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("create_note", "1.0.0", "Create a new note for the user").
		AddStringParameter("user_id", "The user ID who will own the note", true).
		AddStringParameter("content", "The note content", true).
		AddStringParameter("project_id", "Optional project ID to associate with the note", false).
		AddArrayParameter("tags", "Note tags for organization", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			userID, ok := args["user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user_id parameter is required and must be a string")
			}

			content, ok := args["content"].(string)
			if !ok {
				return nil, fmt.Errorf("content parameter is required and must be a string")
			}

			var tags []string
			if t, exists := args["tags"]; exists {
				if tagArray, ok := t.([]interface{}); ok {
					for _, tag := range tagArray {
						if tagStr, ok := tag.(string); ok {
							tags = append(tags, tagStr)
						}
					}
				}
			}

			// Create note request
			createReq := note.CreateNoteRequest{
				Content: content,
				Tags:    tags,
			}

			// Add project ID if provided
			if projectID, exists := args["project_id"]; exists {
				if projIDStr, ok := projectID.(string); ok {
					createReq.ProjectID = &projIDStr
				}
			}

			noteResp, err := deps.NoteService.CreateNote(ctx, userID, createReq)
			if err != nil {
				return nil, fmt.Errorf("failed to create note: %w", err)
			}

			return map[string]any{
				"note":    noteResp,
				"success": true,
				"message": "Note created successfully",
			}, nil
		}).
		AddTags("note", "create", "new").
		Build()
}

// NoteUpdateToolBuilder builds a tool to update existing notes
type NoteUpdateToolBuilder struct{}

func (n *NoteUpdateToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("update_note", "1.0.0", "Update an existing note's information").
		AddStringParameter("user_id", "The user ID who owns the note", true).
		AddStringParameter("note_id", "The note ID to update", true).
		AddStringParameter("content", "New note content", false).
		AddArrayParameter("tags", "New note tags", false).
		AddStringParameter("project_id", "New project ID to associate with note", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			userID, ok := args["user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user_id parameter is required and must be a string")
			}

			noteID, ok := args["note_id"].(string)
			if !ok {
				return nil, fmt.Errorf("note_id parameter is required and must be a string")
			}

			// Build update request with only provided fields
			updateReq := note.UpdateNoteRequest{}

			if content, exists := args["content"]; exists {
				if contentStr, ok := content.(string); ok {
					updateReq.Content = &contentStr
				}
			}

			if projectID, exists := args["project_id"]; exists {
				if projIDStr, ok := projectID.(string); ok {
					updateReq.ProjectID = &projIDStr
				}
			}

			if tags, exists := args["tags"]; exists {
				if tagArray, ok := tags.([]interface{}); ok {
					var tagStrings []string
					for _, tag := range tagArray {
						if tagStr, ok := tag.(string); ok {
							tagStrings = append(tagStrings, tagStr)
						}
					}
					updateReq.Tags = &tagStrings
				}
			}

			noteResp, err := deps.NoteService.UpdateNote(ctx, userID, noteID, updateReq)
			if err != nil {
				return nil, fmt.Errorf("failed to update note: %w", err)
			}

			return map[string]any{
				"note":    noteResp,
				"success": true,
				"message": "Note updated successfully",
			}, nil
		}).
		AddTags("note", "update", "modify").
		Build()
}

// NoteSearchToolBuilder builds a tool to search notes by content
type NoteSearchToolBuilder struct{}

func (n *NoteSearchToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("search_notes", "1.0.0", "Search notes by content and title").
		AddStringParameter("user_id", "The user ID to search notes for", true).
		AddStringParameter("query", "Search query to match against note content and titles", true).
		AddStringParameter("project_id", "Optional project ID to limit search scope", false).
		AddNumberParameter("limit", "Maximum number of notes to return", false).
		AddNumberParameter("offset", "Number of notes to skip for pagination", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			userID, ok := args["user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user_id parameter is required and must be a string")
			}

			query, ok := args["query"].(string)
			if !ok {
				return nil, fmt.Errorf("query parameter is required and must be a string")
			}

			// Set defaults
			limit := 20
			offset := 0

			if l, exists := args["limit"]; exists {
				if limitFloat, ok := l.(float64); ok {
					limit = int(limitFloat)
				}
			}

			if o, exists := args["offset"]; exists {
				if offsetFloat, ok := o.(float64); ok {
					offset = int(offsetFloat)
				}
			}

			// Create search request
			searchReq := note.ListNotesRequest{
				Search: query,
				Offset: offset,
				Limit:  limit,
			}

			var notes []note.NoteResponse
			var total int64
			var err error

			// Check if filtering by project - use different method if so
			if projectID, exists := args["project_id"]; exists {
				if projIDStr, ok := projectID.(string); ok {
					notes, total, err = deps.NoteService.GetProjectNotes(ctx, userID, projIDStr, offset, limit)
					if err != nil {
						return nil, fmt.Errorf("failed to search project notes: %w", err)
					}
					// Filter by search query if provided (manual filtering for project-specific search)
					if query != "" {
						var filteredNotes []note.NoteResponse
						for _, n := range notes {
							if containsIgnoreCase(n.Content, query) {
								filteredNotes = append(filteredNotes, n)
							}
						}
						notes = filteredNotes
						total = int64(len(filteredNotes))
					}
				}
			} else {
				notes, total, err = deps.NoteService.ListUserNotes(ctx, userID, searchReq)
				if err != nil {
					return nil, fmt.Errorf("failed to search notes: %w", err)
				}
			}

			return map[string]any{
				"notes":       notes,
				"total_count": total,
				"returned":    len(notes),
				"offset":      offset,
				"limit":       limit,
				"query":       query,
			}, nil
		}).
		AddTags("note", "search", "content").
		Build()
}
