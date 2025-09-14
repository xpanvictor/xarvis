package catalog

import (
	"context"
	"fmt"

	"github.com/xpanvictor/xarvis/internal/domains/project"
	"github.com/xpanvictor/xarvis/internal/tools"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

// ProjectListToolBuilder builds a tool to fetch user's projects
type ProjectListToolBuilder struct{}

func (p *ProjectListToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("fetch_projects", "1.0.0", "Fetch all projects for the current user").
		AddNumberParameter("limit", "Maximum number of projects to return", false).
		AddNumberParameter("offset", "Number of projects to skip for pagination", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Extract user ID from injected context
			userID, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available")
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

			// Create list request
			listReq := project.ListProjectsRequest{
				Offset: offset,
				Limit:  limit,
			}

			projects, total, err := deps.ProjectService.ListUserProjects(ctx, userID, listReq)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch projects: %w", err)
			}

			return map[string]any{
				"projects":    projects,
				"total_count": total,
				"returned":    len(projects),
				"offset":      offset,
				"limit":       limit,
			}, nil
		}).
		AddTags("project", "list", "user").
		Build()
}

// ProjectInfoToolBuilder builds a tool to get detailed project information
type ProjectInfoToolBuilder struct{}

func (p *ProjectInfoToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("fetch_project_info", "1.0.0", "Get detailed information about a specific project including its notes").
		AddStringParameter("project_id", "The project ID to fetch information for", true).
		AddBooleanParameter("include_notes", "Whether to include project notes", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Extract user ID from injected context
			userID, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available")
			}

			projectID, ok := args["project_id"].(string)
			if !ok {
				return nil, fmt.Errorf("project_id parameter is required and must be a string")
			}

			includeNotes := false
			if inc, exists := args["include_notes"]; exists {
				if incBool, ok := inc.(bool); ok {
					includeNotes = incBool
				}
			}

			// Fetch project details
			projectResp, err := deps.ProjectService.GetProject(ctx, userID, projectID)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch project: %w", err)
			}

			result := map[string]any{
				"project": projectResp,
			}

			// Fetch notes if requested
			if includeNotes {
				notes, _, err := deps.NoteService.GetProjectNotes(ctx, userID, projectID, 0, 100)
				if err != nil {
					deps.Logger.Error("Failed to fetch project notes: %v", err)
					result["notes_error"] = err.Error()
				} else {
					result["notes"] = notes
					result["notes_count"] = len(notes)
				}
			}

			return result, nil
		}).
		AddTags("project", "info", "details").
		Build()
}

// ProjectCreateToolBuilder builds a tool to create new projects
type ProjectCreateToolBuilder struct{}

func (p *ProjectCreateToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("create_project", "1.0.0", "Create a new project for the user").
		AddStringParameter("title", "The project title", true).
		AddStringParameter("description", "The project description", false).
		AddArrayParameter("tags", "Project tags for organization", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Extract user ID from injected context
			userID, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available")
			}

			title, ok := args["title"].(string)
			if !ok {
				return nil, fmt.Errorf("title parameter is required and must be a string")
			}

			description := ""
			if desc, exists := args["description"]; exists {
				if descStr, ok := desc.(string); ok {
					description = descStr
				}
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

			// Create project request
			createReq := project.CreateProjectRequest{
				Name:        title,
				Description: description,
				Tags:        tags,
			}

			projectResp, err := deps.ProjectService.CreateProject(ctx, userID, createReq)
			if err != nil {
				return nil, fmt.Errorf("failed to create project: %w", err)
			}

			return map[string]any{
				"project": projectResp,
				"success": true,
				"message": fmt.Sprintf("Project '%s' created successfully", title),
			}, nil
		}).
		AddTags("project", "create", "new").
		Build()
}

// ProjectUpdateToolBuilder builds a tool to update existing projects
type ProjectUpdateToolBuilder struct{}

func (p *ProjectUpdateToolBuilder) Build(deps *tools.ToolDependencies) (toolsystem.Tool, error) {
	return toolsystem.NewToolBuilder("update_project", "1.0.0", "Update an existing project's information").
		AddStringParameter("project_id", "The project ID to update", true).
		AddStringParameter("title", "New project title", false).
		AddStringParameter("description", "New project description", false).
		AddArrayParameter("tags", "New project tags", false).
		AddStringParameter("status", "New project status", false).
		SetHandler(func(ctx context.Context, args map[string]any) (map[string]any, error) {
			// Extract user ID from injected context
			userID, ok := args["__user_id"].(string)
			if !ok {
				return nil, fmt.Errorf("user context not available")
			}

			projectID, ok := args["project_id"].(string)
			if !ok {
				return nil, fmt.Errorf("project_id parameter is required and must be a string")
			}

			// Build update request with only provided fields
			updateReq := project.UpdateProjectRequest{}

			if title, exists := args["title"]; exists {
				if titleStr, ok := title.(string); ok {
					updateReq.Name = &titleStr
				}
			}

			if desc, exists := args["description"]; exists {
				if descStr, ok := desc.(string); ok {
					updateReq.Description = &descStr
				}
			}

			if status, exists := args["status"]; exists {
				if statusStr, ok := status.(string); ok {
					projectStatus := project.ProjectStatus(statusStr)
					updateReq.Status = &projectStatus
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

			projectResp, err := deps.ProjectService.UpdateProject(ctx, userID, projectID, updateReq)
			if err != nil {
				return nil, fmt.Errorf("failed to update project: %w", err)
			}

			return map[string]any{
				"project": projectResp,
				"success": true,
				"message": "Project updated successfully",
			}, nil
		}).
		AddTags("project", "update", "modify").
		Build()
}
