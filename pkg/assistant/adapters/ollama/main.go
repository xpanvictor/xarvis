package ollama

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	"github.com/xpanvictor/xarvis/pkg/assistant/providers/ollama"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
)

type ollamaAdapter struct {
	op        *ollama.OllamaProvider
	msgBuffer []adapters.ContractResponseDelta
	cfg       adapters.ContractLLMCfg
	drc       *adapters.ContractResponseChannel
}

// DrainBuffer implements adapters.ContractAdapter.
func (o *ollamaAdapter) DrainBuffer(ch adapters.ContractResponseChannel) bool {
	if len(o.msgBuffer) == 0 {
		return false
	}
	// send a copy to avoid races/mutation after send
	snapshot := make([]adapters.ContractResponseDelta, len(o.msgBuffer))
	copy(snapshot, o.msgBuffer)
	select {
	case ch <- snapshot:
		// reset buffer length to zero, keep capacity
		o.msgBuffer = o.msgBuffer[:0]
		return true
	default:
		// channel not ready; keep buffer to retry next tick
		return false
	}
}

// available roles for ollama: system, role, assistant
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
	stream := true
	// construct ollama req
	model := input.HandlerModel
	req := api.ChatRequest{
		Model:    router.GenerateModelName(model),
		Messages: o.ConvertMsgs(input.Msgs),
		Stream:   &stream,
	}

	// construct handler
	var handlerChannel *adapters.ContractResponseChannel
	if rc != nil {
		handlerChannel = rc
	} else if o.drc != nil {
		handlerChannel = o.drc
	} else {
		panic("hndl error: no input channel for ollama adapter provided")
	}
	// per-request sequence counter for ordering
	var seq uint
	var handler api.ChatResponseFunc = func(cr api.ChatResponse) error {

		msg := o.ConvertMsgBackward([]api.ChatResponse{cr})[0]
		// assign monotonically increasing index
		seq++
		msg.Index = seq
		o.msgBuffer = append(o.msgBuffer, msg)
		if msg.Done {
			// signal end of processing
			// no-op; completion is handled after Chat returns
		}
		// todo: error
		return nil
	}

	// Drain buffer periodically in parallel with streaming
	ctx2, cancel := context.WithCancel(ctx)
	bft := time.NewTicker(o.cfg.DeltaTimeDuration)
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for {
			select {
			case <-ctx2.Done():
				return
			case <-bft.C:
				o.DrainBuffer(*handlerChannel)
			}
		}
	}()

	// Stream from provider; returns when complete or on error
	err = o.op.Chat(ctx2, req, handler)
	// stop drainer and wait it exit
	cancel()
	<-drained
	// final flush
	o.DrainBuffer(*handlerChannel)
	// close downstream channel to signal completion
	close(*handlerChannel)

	if err != nil {
		return adapters.ContractResponse{ID: genID, StartedAt: startedAt, Error: err}
	}
	return adapters.ContractResponse{ID: genID, StartedAt: startedAt, Done: true}
}

func New(
	provider *ollama.OllamaProvider,
	cfg adapters.ContractLLMCfg,
	defaultResponseChannel *adapters.ContractResponseChannel,
) adapters.ContractAdapter {
	if cfg.DeltaTimeDuration == 0 {
		// Batch deltas a bit longer to avoid word-by-word streaming
		cfg.DeltaTimeDuration = 150 * time.Millisecond
	}
	if cfg.DeltaBufferLimit == 0 {
		// Allow more tokens per tick before flush
		cfg.DeltaBufferLimit = 24
	}
	return &ollamaAdapter{
		op:  provider,
		cfg: cfg,
		// initialize with zero length and preallocated capacity to avoid nil entries
		msgBuffer: make([]adapters.ContractResponseDelta, 0, int(cfg.DeltaBufferLimit)),
		drc:       defaultResponseChannel,
	}
}
