package gemini

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
	"github.com/xpanvictor/xarvis/pkg/assistant/providers/gemini"
	"github.com/xpanvictor/xarvis/pkg/assistant/router"
)

type geminiAdapter struct {
	gp  *gemini.GeminiProvider
	cfg adapters.ContractLLMCfg
}

// New creates a new GeminiAdapter instance.
func New(
	provider *gemini.GeminiProvider,
	cfg adapters.ContractLLMCfg,
	_ *adapters.ContractResponseChannel, // deprecated, no longer used
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

// Process implements adapters.ContractAdapter.
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
	log.Printf("toools------ %v", input.ToolList)
	handlerChannel := rc

	modelName := router.GenerateModelName(input.HandlerModel)
	model := g.gp.GetModel("gemini-2.5-flash-lite")
	model.Tools = g.ConvertTools(input.ToolList)

	cs := model.StartChat()
	iter := cs.SendMessageStream(ctx, g.ConvertMsgs(input.Msgs)...)

	requestMsgBuffer := make([]adapters.ContractResponseDelta, 0, int(g.cfg.DeltaBufferLimit))
	var seq uint

	ctx2, cancel := context.WithCancel(ctx)
	bft := time.NewTicker(g.cfg.DeltaTimeDuration)
	drained := make(chan struct{})

	drainRequestBuffer := func(ch adapters.ContractResponseChannel) bool {
		if len(requestMsgBuffer) == 0 {
			return false
		}
		snapshot := make([]adapters.ContractResponseDelta, len(requestMsgBuffer))
		copy(snapshot, requestMsgBuffer)
		select {
		case <-ctx.Done():
			return false
		case ch <- snapshot:
			requestMsgBuffer = requestMsgBuffer[:0]
			return true
		default:
			return false
		}
	}

	go func() {
		defer close(drained)
		for {
			select {
			case <-ctx2.Done():
				return
			case <-bft.C:
				drainRequestBuffer(*handlerChannel)
			}
		}
	}()

	err = g.gp.Chat(ctx2, modelName, iter, func(resp *genai.GenerateContentResponse) error {
		deltas := g.ConvertMsgBackward(resp)
		for _, msg := range deltas {
			seq++
			msg.Index = seq
			requestMsgBuffer = append(requestMsgBuffer, msg)
		}
		return nil
	})

	cancel()
	<-drained
	drainRequestBuffer(*handlerChannel)

	// Send final "Done" message
	requestMsgBuffer = append(requestMsgBuffer, adapters.ContractResponseDelta{
		Done:      true,
		CreatedAt: time.Now(),
	})
	drainRequestBuffer(*handlerChannel)

	close(*handlerChannel)

	if err != nil {
		return adapters.ContractResponse{ID: genID, StartedAt: startedAt, Error: fmt.Errorf("Gemini chat failed: %w", err)}
	}
	return adapters.ContractResponse{ID: genID, StartedAt: startedAt, Done: true}
}

// ConvertMsgs converts ContractMessage to Gemini's genai.Content.
func (g *geminiAdapter) ConvertMsgs(msgs []adapters.ContractMessage) []genai.Part {
	var parts []genai.Part
	for _, msg := range msgs {
		// Gemini handles history differently; we construct a flat list of parts.
		// Role is inferred by position or can be set on Content.
		// For simplicity, we'll just send the text content.
		content := fmt.Sprintf("%s: %s (at %s)", msg.Role, msg.Content, msg.CreatedAt.Local().Format(time.RFC3339))
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
			if txt, ok := part.(genai.Text); ok {
				textMsg += string(txt)
			}
			if fc, ok := part.(genai.FunctionCall); ok {
				toolCalls = append(toolCalls, adapters.ContractToolCall{
					ID:        uuid.New(),
					CreatedAt: createdAt,
					ToolName:  fc.Name,
					Arguments: fc.Args,
				})
			}
		}

		deltas = append(deltas, adapters.ContractResponseDelta{
			Msg: &adapters.ContractMessage{
				Content:   textMsg,
				Role:      adapters.ASSISTANT,
				CreatedAt: createdAt,
			},
			ToolCalls: &toolCalls,
			Done:      false,
			CreatedAt: createdAt,
		})
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
				schemaType = genai.TypeString // default to string
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
