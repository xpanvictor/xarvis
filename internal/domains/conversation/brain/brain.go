package brain

import (
	"context"
	"sync"
	"time"

	"github.com/xpanvictor/xarvis/internal/config"
	"github.com/xpanvictor/xarvis/internal/domains/conversation"
	"github.com/xpanvictor/xarvis/pkg/assistant"
	toolsystem "github.com/xpanvictor/xarvis/pkg/tool_system"
)

type Brain struct {
	cfg       config.BrainConfig // change to brain configs
	assistant assistant.Assistant
	registry  toolsystem.Registry
	executor  toolsystem.Executor
	// memory
	// msg history
}

// Brain Decision System using Messages (BDSM)
// when the brain gets a percept
// it enters a thinking <-> acting loop
// thinking -> digest(prev_information) from memory
//
//	-> draw inferences
//	-> consider adding to memory
//
// acting -> parallel actions
// will be using a Queue for this
func (b *Brain) Decide(ctx context.Context, msg conversation.Message) (conversation.Message, error) {
	asstInputQueue := make([]assistant.AssistantInput, 0)
	asstInputQueue = append(asstInputQueue,
		assistant.NewAssistantInput(
			[]assistant.AssistantMessage{
				msg.ToAssistantMessage(),
			},
			"",
		),
	)

	toolCallsCount := 0
	var resp assistant.AssistantOutput

	// the loop structure
	for len(asstInputQueue) > 0 {
		currAsstInput := asstInputQueue[0]
		asstInputQueue = asstInputQueue[1:]
		resp, err := b.assistant.ProcessPrompt(
			ctx,
			currAsstInput,
		)
		if err != nil {
			panic("unimplemented brain decision err")
		}
		// handle tools
		toolMsgs, callLength := b.ExecuteToolCallsParallel(ctx, resp.ToolCalls)

		if toolCallsCount >= b.cfg.MaxToolCallLimit {

		} else {
			toolCallsCount += callLength
		}

		// construct next asst input if necessary
		if callLength > 0 {
			// pass in all tool msgs
			asstInput := assistant.NewAssistantInput(
				toolMsgs,
				"Tool call results",
			)
			asstInputQueue = append(asstInputQueue, asstInput)
		}
	}

	return conversation.AssistantMsgToMessage(&resp.Response, msg.UserId), nil
}

func (b *Brain) ExecuteToolCallsParallel(
	ctx context.Context,
	toolCalls []assistant.ToolCall,
) ([]assistant.AssistantMessage, int) {
	toolResponses := make([]assistant.AssistantMessage, 0)
	// mutex & ...
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	for _, toolCall := range toolCalls {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// call fn
			callResp, _, err := b.executor.Execute(ctx, b.registry, toolCall)
			var toolResponse assistant.AssistantMessage
			if err != nil {
				toolResponse = assistant.AssistantMessage{
					MsgRole:      assistant.TOOL,
					Content:      "Failed tool",
					CreatedAt:    time.Now(),
					ToolResponse: callResp,
				}
			} else {
				toolResponse = assistant.AssistantMessage{
					MsgRole:      assistant.TOOL,
					Content:      "Tool result",
					CreatedAt:    time.Now(),
					ToolResponse: callResp,
				}
			}
			mu.Lock()
			toolResponses = append(toolResponses, toolResponse)
			mu.Unlock()
		}()
	}
	wg.Wait()
	return toolResponses, len(toolCalls)
}
