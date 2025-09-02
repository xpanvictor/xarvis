package router

import (
	"context"
	"fmt"

	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
)

// todo: new
// Takes in providers
// Returns Multiplexer
func New() Mux {
	return Mux{}
}

func GenerateModelName(m adapters.ContractSelectedModel) string {
	return fmt.Sprintf("%v%v", m.Name, m.Version)
}

func (m *Mux) Stream(
	ctx context.Context,
	input adapters.ContractInput,
	rc *adapters.ContractResponseChannel,
) {
	sm := m.RouterPolicy.Select(input)
	// handle input
	handlerAdapterPack := m.AdapterMap[GenerateModelName(sm)]
	input.HandlerModel = sm
	out := handlerAdapterPack.Adapter.Process(ctx, input, rc)
	if out.Error != nil {
		panic("unimpl error here")
	}
	// log user req handled
}
