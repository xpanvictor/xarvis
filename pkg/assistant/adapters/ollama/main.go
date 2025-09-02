package ollama

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	"github.com/xpanvictor/xarvis/pkg/assistant/providers/ollama"
)

type ollamaAdapter struct {
	op        ollama.OllamaProvider
	msgBuffer []adapters.ContractResponseDelta
	cfg       adapters.ContractLLMCfg
	drc       *adapters.ContractResponseChannel
}

// DrainBuffer implements adapters.ContractAdapter.
func (o *ollamaAdapter) DrainBuffer(ch adapters.ContractResponseChannel) bool {
	ch <- o.msgBuffer
	o.msgBuffer = make([]adapters.ContractResponseDelta, o.cfg.DeltaBufferLimit)
	return true
}

func (o ollamaAdapter) ConvertMsgs(msgs []adapters.ContractMessage) []api.Message {
	var convertedMsgs []api.Message
	for _, msg := range msgs {
		convertedMsgs = append(
			convertedMsgs, api.Message{
				Role: string(msg.Role),
				// time aware system
				Content: fmt.Sprintf("%v\nCurrent Time: %v", msg.Content, msg.CreatedAt.Local().String()),
			},
		)
	}
	return convertedMsgs
}

func (o ollamaAdapter) ConvertMsgBackward(msgs []api.ChatResponse) []adapters.ContractResponseDelta {
	var cm []adapters.ContractResponseDelta
	for _, msg := range msgs {
		// extract msg
		textMsg := msg.Message.Content
		role := adapters.MsgRole(msg.Message.Role)
		rawToolCalls := msg.Message.ToolCalls
		toolCalls := make([]adapters.ContractToolCall, 0)
		// todo: map toolcall
		fmt.Printf("%v", rawToolCalls)
		done := msg.Done
		createdAt := msg.CreatedAt
		// extract tools
		cm = append(
			cm,
			adapters.ContractResponseDelta{
				Msg:       &adapters.ContractMessage{Content: textMsg, Role: role, CreatedAt: createdAt},
				ToolCalls: &toolCalls,
				Done:      done,
				CreatedAt: createdAt,
			},
		)
	}
	return cm
}

// Process implements adapters.ContractAdapter.
func (o *ollamaAdapter) Process(ctx context.Context, input adapters.ContractInput, rc *adapters.ContractResponseChannel) adapters.ContractResponse {
	genID, err := uuid.NewUUID()
	if err != nil {
		panic("unimpl")
	}
	startedAt := time.Now()
	// construct ollama req
	model := input.HandlerModel
	req := api.ChatRequest{
		Model:    fmt.Sprintf("%v%v", model.Name, model.Version),
		Messages: o.ConvertMsgs(input.Msgs),
		// Stream:   true,
	}
	// construct handler
	var handlerChannel *adapters.ContractResponseChannel
	if rc != nil {
		handlerChannel = rc
	} else if o.drc != nil {
		handlerChannel = o.drc
	} else {
		panic("no input channel for ollama adapter provided")
	}
	var handler api.ChatResponseFunc = func(cr api.ChatResponse) error {
		msg := o.ConvertMsgBackward([]api.ChatResponse{cr})[0]
		o.msgBuffer = append(o.msgBuffer, msg)
		if msg.Done {
			// signal end of processing
			ctx.Done()
		}
		// todo: error
		return nil
	}

	err = o.op.Chat(ctx, req, handler)
	if err != nil {
		return adapters.ContractResponse{
			ID:        genID,
			StartedAt: startedAt,
			Error:     err,
		}
	}

	// rbt := time.NewTicker(o.cfg.RebounceDuration)
	bft := time.NewTicker(o.cfg.DeltaTimeDuration)
	for {
		select {
		case <-ctx.Done():
			return adapters.ContractResponse{
				ID:        genID,
				StartedAt: startedAt,
				Done:      true,
			}
		case <-bft.C:
			o.DrainBuffer(*handlerChannel)
		}
	}
}

func New(
	provider ollama.OllamaProvider,
	cfg adapters.ContractLLMCfg,
	defaultResponseChannel *adapters.ContractResponseChannel,
) adapters.ContractAdapter {
	return &ollamaAdapter{
		op:        provider,
		cfg:       cfg,
		msgBuffer: make([]adapters.ContractResponseDelta, cfg.DeltaBufferLimit),
		drc:       defaultResponseChannel,
	}
}
