package adapters

import (
	"context"
	"time"
)

type ContractLLMCfg struct {
	DeltaBufferLimit  uint
	DeltaTimeDuration time.Duration
	RebounceDuration  time.Duration
}

type ContractAdapter interface {
	// while
	// 	ctx not done & buffer not full & !bufferTimeout
	// 	buffer stream else insert into stream
	// but controlled
	// process should be time aware
	Process(
		ctx context.Context,
		input ContractInput,
		// allows default response channel
		responseChannel *ContractResponseChannel,
	) ContractResponse

	DrainBuffer(ch ContractResponseChannel) bool
}
