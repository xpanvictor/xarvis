package toolsystem

import (
	"fmt"
	"sync"
)

type Tool struct {
	Spec    ToolSpec
	Handler ToolHandler
}

type Registry interface {
	Register(t Tool) error
	Unregister(toolId string) error
	Get(toolId string) (Tool, bool)
	List() []Tool
}

type memoryRegistry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// Get implements Registry.
func (m *memoryRegistry) Get(toolId string) (Tool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	tool, exist := m.tools[toolId]
	return tool, exist
}

// List implements Registry.
func (m *memoryRegistry) List() []Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		out = append(out, tool)
	}
	return out
}

// Register implements Registry.
func (m *memoryRegistry) Register(t Tool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	toolId := GetToolId(t)
	if _, exists := m.tools[toolId]; exists {
		return fmt.Errorf("tool with id %s exists", toolId)
	}
	m.tools[toolId] = t
	return nil
}

// Unregister implements Registry.
func (m *memoryRegistry) Unregister(toolId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// allow multiple unregister
	delete(m.tools, toolId)
	return nil
}

func NewMemoryRegistry() Registry {
	return &memoryRegistry{
		tools: make(map[string]Tool),
	}
}

func GetToolId(t Tool) string {
	internal_mask := "xp_t"
	return fmt.Sprintf("%s:%s:%s", internal_mask, t.Spec.Name, t.Spec.Version)
}
