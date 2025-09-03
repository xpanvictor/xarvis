package router

import (
	"context"
	"fmt"

	"github.com/xpanvictor/xarvis/pkg/assistant/adapters"
)

type DefaultRP struct{}

func (*DefaultRP) Select(input adapters.ContractInput) adapters.ContractSelectedModel {
	return adapters.ContractSelectedModel{
		Name:    "llama",
		Version: "3:8b",
	}
}

// todo: new
// Takes in adapters
// Returns Multiplexer
func New(
	ads []adapters.ContractAdapter,
) Mux {
	adm := make(map[string]AdapterPack)
	drp := &DefaultRP{}
	dn := adapters.ContractSelectedModel{
		Name:    "llama",
		Version: "3:8b",
	}

	for _, ad := range ads {
		name := GenerateModelName(dn)
		adm[name] = AdapterPack{
			Name:         name,
			Adapter:      ad,
			DefaultModel: dn,
		}
	}

	return Mux{
		RouterPolicy: drp,
		AdapterMap:   adm,
	}
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
	input.HandlerModel = sm
	out := m.AdapterMap[GenerateModelName(sm)].Adapter.Process(ctx, input, rc)
	if out.Error != nil {
		panic("unimpl error here")
	}
	// log user req handled
}
