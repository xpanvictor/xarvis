package processor

import "context"

// Processor interface for internal system decision making
type Processor interface {
	// Process takes an instruction, input data, and returns a structured response
	Process(ctx context.Context, instruction string, input interface{}) (interface{}, error)

	// ProcessWithType takes an instruction, input data, and expected response type
	ProcessWithType(ctx context.Context, instruction string, input interface{}, responseType interface{}) error
}
