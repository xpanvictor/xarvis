package brain

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/pkg/assistant"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
)

// Example usage of the updated Brain with contract types

// ExampleBrainUsage demonstrates how to use the Brain with contract types
func ExampleBrainUsage() {
	// This example shows how the new Brain system works with contract types
	// Note: This is for demonstration purposes and requires proper setup

	ctx := context.Background()

	// Create a sample user message
	userMessage := conversation.Message{
		Id:        uuid.New().String(),
		UserId:    "user123",
		Text:      "What's the weather like in New York?",
		Timestamp: time.Now(),
		MsgRole:   assistant.USER,
		Tags:      []string{"weather", "query"},
	}

	// The Brain would be set up with:
	// 1. Tool registry containing weather tools
	// 2. Executor for running tools
	// 3. Contract adapter (e.g., Ollama adapter)
	// 4. Configuration

	// Example flow:
	// brain := SetupBrainWithOllama(cfg, ollamaProvider, logger)
	// response, err := brain.Decide(ctx, userMessage)

	// The Decide method now:
	// 1. Converts conversation.Message to adapters.ContractMessage
	// 2. Gets available tools from registry
	// 3. Processes through contract adapter with tool support
	// 4. Executes any tool calls via ExecuteToolCallsParallel
	// 5. Converts response back to conversation.Message

	_ = ctx
	_ = userMessage
}

// ExampleToolExecution shows how tool execution works
func ExampleToolExecution() {
	// Example of tool calls being executed:

	ctx := context.Background()

	// Sample tool calls from LLM
	toolCalls := []adapters.ContractToolCall{
		{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			ToolName:  "get_weather",
			Arguments: map[string]any{
				"location":         "New York",
				"units":            "celsius",
				"include_forecast": true,
			},
		},
	}

	// Brain.ExecuteToolCallsParallel would:
	// 1. Look up each tool in the registry
	// 2. Execute tools in parallel using goroutines
	// 3. Convert results back to conversation.Message format
	// 4. Return tool response messages for the conversation

	_ = ctx
	_ = toolCalls
}

// Message flow example:
// 1. User sends: "What's the weather in NYC?"
// 2. Brain.Decide converts to contract format
// 3. LLM processes and decides to call get_weather tool
// 4. Brain executes tool call via ExecuteToolCallsParallel
// 5. Tool returns weather data
// 6. Brain sends tool result back to LLM
// 7. LLM generates natural language response
// 8. Brain converts back to conversation.Message
// 9. User receives: "The weather in New York City is currently 22°C and sunny..."
