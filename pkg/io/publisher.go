package io

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
)

type Publisher struct {
	reg registry.DeviceRegistry
}

func New(reg registry.DeviceRegistry) Publisher {
	return Publisher{reg: reg}
}

func (p *Publisher) SendTextDelta(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
	seq int,
	text string,
) error {

	if eps, ok := p.reg.FetchTextFanoutEndpoint(userID); ok {
		for _, ep := range eps {
			_ = ep.SendTextDelta(sessionID, seq, text)
			// todo: emit error for text misses
		}
		return nil
	}
	// emit text broadcast failed event
	return fmt.Errorf("couldn't broadcast text")
}

func (p *Publisher) SendAudioFrame(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
	frame []byte,
) error {
	ep, ok := p.reg.SelectEndpointWithMRU(userID)
	if !ok || !ep.IsAlive() || !ep.Caps().AudioSink {
		return fmt.Errorf("couldn't send audio frame")
	}
	return ep.SendAudioFrame(sessionID, frame)
}

func (p *Publisher) SendEvent(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
	name string,
	payload any,
) error {
	eps := p.reg.ListUserEndpoints(userID)
	for _, ep := range eps {
		if ep.IsAlive() {
			ep.SendEvent(sessionID, name, payload)
		}
	}
	return nil
}
