package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	"github.com/xpanvictor/xarvis/pkg/assistant/providers/gemini"
)

type geminiAdapter struct {
	gp  *gemini.GeminiProvider
	cfg adapters.ContractLLMCfg
}

// New creates a new GeminiAdapter instance.
func New(
	provider *gemini.GeminiProvider,
	cfg adapters.ContractLLMCfg,
	_ *adapters.ContractResponseChannel, // deprecated
) adapters.ContractAdapter {
	if cfg.DeltaTimeDuration == 0 {
		cfg.DeltaTimeDuration = 150 * time.Millisecond
	}
	if cfg.DeltaBufferLimit == 0 {
		cfg.DeltaBufferLimit = 24
	}
	return &geminiAdapter{
		gp:  provider,
		cfg: cfg,
	}
}

// DrainBuffer is a no-op since we don't use shared buffers.
func (g *geminiAdapter) DrainBuffer(ch adapters.ContractResponseChannel) bool {
	return false
}

// Process streams deltas, then guarantees: (1) flusher stopped, (2) buffer drained, (3) Done sent, (4) channel closed once.
func (g *geminiAdapter) Process(ctx context.Context, input adapters.ContractInput, rc *adapters.ContractResponseChannel) adapters.ContractResponse {
	genID, err := uuid.NewUUID()
	if err != nil {
		return adapters.ContractResponse{Error: fmt.Errorf("failed to generate UUID: %w", err)}
	}
	startedAt := time.Now()

	if rc == nil {
		return adapters.ContractResponse{
			ID:        genID,
			StartedAt: startedAt,
			Error:     fmt.Errorf("response channel is required"),
		}
	}
	handlerChannel := rc

	model := g.gp.GetModel("gemini-2.5-flash-lite")
	model.Tools = g.ConvertTools(input.ToolList)

	log.Printf("Gemini: Model configured with %d tools", len(model.Tools))

	cs := model.StartChat()
	convertedMsgs := g.ConvertMsgs(input.Msgs)
	log.Printf("Gemini: Converted %d input messages to %d parts", len(input.Msgs), len(convertedMsgs))

	iter := cs.SendMessageStream(ctx, convertedMsgs...)

	requestMsgBuffer := make([]adapters.ContractResponseDelta, 0, int(g.cfg.DeltaBufferLimit))
	var seq uint

	ctx2, cancel := context.WithCancel(ctx)
	log.Printf("Gemini: Created context for processing")
	ticker := time.NewTicker(g.cfg.DeltaTimeDuration)
	flusherDone := make(chan struct{})

	// Helper: drain local buffer to channel (non-blocking on receiver side beyond the send).
	drainRequestBuffer := func(ch adapters.ContractResponseChannel) bool {
		if len(requestMsgBuffer) == 0 {
			return false
		}
		snapshot := make([]adapters.ContractResponseDelta, len(requestMsgBuffer))
		copy(snapshot, requestMsgBuffer)

		select {
		case <-ctx2.Done():
			return false
		case ch <- snapshot:
			requestMsgBuffer = requestMsgBuffer[:0]
			return true
		default:
			// Receiver not ready right now; skip (next tick will try again)
			return false
		}
	}

	// Periodic flusher goroutine
	go func() {
		defer close(flusherDone)
		for {
			select {
			case <-ctx2.Done():
				return
			case <-ticker.C:
				// Best effort; if receiver is not ready, we try on the next tick
				_ = drainRequestBuffer(*handlerChannel)
			}
		}
	}()

	// Collect deltas from Gemini into our local buffer
	log.Printf("Gemini: Starting Chat call with %d messages", len(input.Msgs))
	err = g.gp.Chat(ctx2, "", iter, func(resp *genai.GenerateContentResponse) error {
		log.Printf("Gemini: Received response with %d candidates", len(resp.Candidates))
		deltas := g.ConvertMsgBackward(resp)
		log.Printf("Gemini: Converted to %d deltas", len(deltas))
		for _, msg := range deltas {
			seq++
			msg.Index = seq
			requestMsgBuffer = append(requestMsgBuffer, msg)
		}
		return nil
	})

	if err != nil {
		log.Printf("Gemini: Chat error: %v", err)
	} else {
		log.Printf("Gemini: Chat completed successfully, total buffer size: %d", len(requestMsgBuffer))
	}

	// Begin shutdown sequence in strict order:
	// 1) Stop new flush ticks, 2) do a final flush, 3) cancel ctx2, 4) wait flusher exit, 5) send Done, 6) close channel.
	ticker.Stop()

	// Final drain: ensure all buffered deltas are sent before cancelling context
	log.Printf("Gemini: Final drain - buffer size before: %d", len(requestMsgBuffer))
	flushed := drainRequestBuffer(*handlerChannel)
	log.Printf("Gemini: Final drain completed, flushed: %v", flushed)

	// Now cancel context and wait for flusher to exit
	cancel()
	<-flusherDone

	// Send final Done (block a little to guarantee delivery, but avoid permanent deadlock)
	finalDelta := []adapters.ContractResponseDelta{{
		Done:      true,
		CreatedAt: time.Now(),
	}}

	log.Printf("Gemini: Sending Done signal")

	// Small timeout context only for the final Done send
	sendCtx, sendCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer sendCancel()

	select {
	case <-sendCtx.Done():
		// Timeout — log and continue to close to avoid leaking
		log.Printf("gemini: timeout sending final Done; closing channel anyway")
	case *handlerChannel <- finalDelta:
		// Done delivered
	}

	// Single point of closure: the adapter owns channel close
	close(*handlerChannel)

	if err != nil {
		return adapters.ContractResponse{ID: genID, StartedAt: startedAt, Error: fmt.Errorf("gemini chat failed: %w", err)}
	}
	return adapters.ContractResponse{ID: genID, StartedAt: startedAt, Done: true}
}

// ConvertMsgs converts ContractMessage to Gemini's genai.Content.
func (g *geminiAdapter) ConvertMsgs(msgs []adapters.ContractMessage) []genai.Part {
	var parts []genai.Part
	for _, msg := range msgs {
		content := fmt.Sprintf("[meta: msg sent at %v from %v] %v\n", msg.CreatedAt.Local(), msg.Role, msg.Content)
		parts = append(parts, genai.Text(content))
	}
	return parts
}

// ConvertMsgBackward converts Gemini's response to ContractResponseDelta.
func (g *geminiAdapter) ConvertMsgBackward(resp *genai.GenerateContentResponse) []adapters.ContractResponseDelta {
	var deltas []adapters.ContractResponseDelta
	if resp == nil {
		return deltas
	}

	createdAt := time.Now()

	for _, cand := range resp.Candidates {
		if cand.Content == nil {
			continue
		}

		var textMsg string
		var toolCalls []adapters.ContractToolCall

		for _, part := range cand.Content.Parts {
			// 1) Direct function calls
			if fc, ok := part.(genai.FunctionCall); ok {
				toolCalls = append(toolCalls, adapters.ContractToolCall{
					ID:        uuid.New(),
					CreatedAt: createdAt,
					ToolName:  fc.Name,
					Arguments: fc.Args,
				})
				continue
			}

			// 2) Text (may also contain JSON-encoded tool call)
			if txt, ok := part.(genai.Text); ok {
				textStr := string(txt)
				textMsg += textStr

				// Try to parse as JSON-encoded tool call (optional)
				var potentialToolCall struct {
					FunctionName string                 `json:"function_name"`
					Arguments    map[string]interface{} `json:"arguments"`
				}
				if len(textStr) > 10 && textStr[0] == '{' && textStr[len(textStr)-1] == '}' {
					if err := json.Unmarshal([]byte(textStr), &potentialToolCall); err == nil && potentialToolCall.FunctionName != "" {
						toolCalls = append(toolCalls, adapters.ContractToolCall{
							ID:        uuid.New(),
							CreatedAt: createdAt,
							ToolName:  potentialToolCall.FunctionName,
							Arguments: potentialToolCall.Arguments,
						})
					}
				}
				continue
			}

			// 3) Unexpected parts
			log.Printf("Encountered unexpected part type: %T, value: %+v", part, part)
		}

		if textMsg != "" || len(toolCalls) > 0 {

			td := adapters.ContractResponseDelta{
				Msg: &adapters.ContractMessage{
					Content:   textMsg,
					Role:      adapters.ASSISTANT,
					CreatedAt: createdAt,
				},
				ToolCalls: toolCalls,
				Done:      false,
				CreatedAt: createdAt,
			}
			deltas = append(deltas, td)
		}

	}

	return deltas
}

// ConvertTools converts our contract tools to Gemini's tool format.
func (g *geminiAdapter) ConvertTools(tools []adapters.ContractTool) []*genai.Tool {
	if len(tools) == 0 {
		return nil
	}

	geminiTools := make([]*genai.Tool, len(tools))
	for i, ct := range tools {
		properties := make(map[string]*genai.Schema)
		for propName, propDef := range ct.ToolFunction.Parameters.Properties {
			var schemaType genai.Type
			switch propDef.Type {
			case "string":
				schemaType = genai.TypeString
			case "integer":
				schemaType = genai.TypeInteger
			case "number":
				schemaType = genai.TypeNumber
			case "boolean":
				schemaType = genai.TypeBoolean
			default:
				schemaType = genai.TypeString
			}

			properties[propName] = &genai.Schema{
				Type:        schemaType,
				Description: propDef.Description,
				Enum:        propDef.Enum,
			}
		}

		geminiTools[i] = &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{
				{
					Name:        ct.Name,
					Description: ct.Description,
					Parameters: &genai.Schema{
						Type:       genai.TypeObject,
						Properties: properties,
						Required:   ct.ToolFunction.RequiredProps,
					},
				},
			},
		}
	}
	return geminiTools
}
