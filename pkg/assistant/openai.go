package assistant

import (
	"context"
	"errors"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/xpanvictor/xarvis/internal/config"
)

type openAIAssistant struct {
	client openai.Client
}

// ProcessPrompt implements Assistant.
func (o openAIAssistant) ProcessPrompt(
	ctx context.Context,
	input AssistantInput,
) (*AssistantOutput, error) {
	convertedMsgs := make([]openai.ChatCompletionMessageParamUnion, 0)
	for _, msg := range input.Msgs {
		convertedMsgs = append(convertedMsgs, convertToOpenaiMsg(msg))
	}
	chatCompletion, err := o.client.Chat.Completions.New(
		ctx,
		openai.ChatCompletionNewParams{
			Messages: convertedMsgs,
			Model:    openai.ChatModelGPT3_5Turbo,
		},
	)
	if err != nil {
		return nil, errors.New("completion failed")
	}
	return &AssistantOutput{
		Id: chatCompletion.ID,
		Response: AssistantMessage{
			Content:   chatCompletion.JSON.Choices.Raw(),
			CreatedAt: time.Now(),
			MsgRole:   ASSISTANT,
		},
		// TODO: tool parsing
		ToolCalls: make([]ToolCall, 0),
	}, nil
}

func convertToOpenaiMsg(msg AssistantMessage) openai.ChatCompletionMessageParamUnion {
	switch msg.MsgRole {
	case ASSISTANT:
		return openai.AssistantMessage(msg.Content)
	case USER:
		return openai.UserMessage(msg.Content)
	case SYSTEM:
		return openai.SystemMessage(msg.Content)
	}
	return openai.UserMessage(msg.Content)
}

func NewAssistant(assistantCfg config.AssistantKeysObj) Assistant {
	return openAIAssistant{
		client: openai.NewClient(
			option.WithAPIKey(
				assistantCfg.OpenAiApiKey,
			),
		),
	}
}
