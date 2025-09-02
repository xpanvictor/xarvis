package router

import "github.com/xpanvictor/xarvis/pkg/assistant/adapters"

type AdapterPack struct {
	Adapter      adapters.ContractAdapter
	Name         string
	DefaultModel adapters.ContractSelectedModel
}

type Mux struct {
	RouterPolicy RoutePolicy
	AdapterMap   map[string]AdapterPack
}

type RoutePolicy interface {
	Select(input adapters.ContractInput) adapters.ContractSelectedModel
}
