package prompts

import (
	"github.com/xpanvictor/xarvis/internal/types"
	"github.com/xpanvictor/xarvis/pkg/assistant"
)

type PromptDefinition struct {
	Content string
	Version float32
}

type SYS_PROMPT struct {
	Intent         string
	CurrentVersion float32
	Items          map[float32]PromptDefinition // version-content
}

func (sp *SYS_PROMPT) GetVersion(version float32) (PromptDefinition, bool) {
	i, ok := sp.Items[version]
	return i, ok
}

func (sp *SYS_PROMPT) GetCurrentPrompt() PromptDefinition {
	return sp.Items[sp.CurrentVersion]
}

func (pd PromptDefinition) ToMessage() types.Message {
	return types.Message{
		MsgRole: assistant.ASSISTANT,
		Text:    pd.Content,
	}
}
