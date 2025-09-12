package prompts

import "github.com/xpanvictor/xarvis/pkg/assistant/adapters"

type PromptDefinition struct {
	Content string
	Version float32
}

type SYS_PROMPT struct {
	Intent         string
	CurrentVersion float32
	Items          map[float32]PromptDefinition // version-content
}

func (pd PromptDefinition) ToContractMessage() adapters.ContractMessage {
	return adapters.ContractMessage{
		Role:    adapters.ASSISTANT,
		Content: pd.Content,
	}
}
