package toolsetup

import (
	"fmt"

	"github.com/xpanvictor/xarvis/internal/tools"
	"github.com/xpanvictor/xarvis/internal/tools/catalog"
)

// RegisterToolBuilders registers all tool builders with the factory
// This function exists in a separate package to avoid import cycles
func RegisterToolBuilders(factory *tools.ToolFactory) error {
	// Project management tools
	if err := factory.RegisterBuilder("project_list", &catalog.ProjectListToolBuilder{}); err != nil {
		return fmt.Errorf("failed to register project list tool: %w", err)
	}

	if err := factory.RegisterBuilder("project_info", &catalog.ProjectInfoToolBuilder{}); err != nil {
		return fmt.Errorf("failed to register project info tool: %w", err)
	}

	if err := factory.RegisterBuilder("project_create", &catalog.ProjectCreateToolBuilder{}); err != nil {
		return fmt.Errorf("failed to register project create tool: %w", err)
	}

	if err := factory.RegisterBuilder("project_update", &catalog.ProjectUpdateToolBuilder{}); err != nil {
		return fmt.Errorf("failed to register project update tool: %w", err)
	}

	// Note management tools
	if err := factory.RegisterBuilder("note_list", &catalog.NoteListToolBuilder{}); err != nil {
		return fmt.Errorf("failed to register note list tool: %w", err)
	}

	if err := factory.RegisterBuilder("note_create", &catalog.NoteCreateToolBuilder{}); err != nil {
		return fmt.Errorf("failed to register note create tool: %w", err)
	}

	if err := factory.RegisterBuilder("note_update", &catalog.NoteUpdateToolBuilder{}); err != nil {
		return fmt.Errorf("failed to register note update tool: %w", err)
	}

	if err := factory.RegisterBuilder("note_search", &catalog.NoteSearchToolBuilder{}); err != nil {
		return fmt.Errorf("failed to register note search tool: %w", err)
	}

	// Web search tools
	if err := factory.RegisterBuilder("web_search", &catalog.WebSearchToolBuilder{}); err != nil {
		return fmt.Errorf("failed to register web search tool: %w", err)
	}

	return nil
}
