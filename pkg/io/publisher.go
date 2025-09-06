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
		// Debug logging to trace message routing
		fmt.Printf("[PUBLISHER] Sending text to UserID: %s, SessionID: %s, EndpointCount: %d, Text: %.50s...\n",
			userID, sessionID, len(eps), text)

		for i, ep := range eps {
			fmt.Printf("[PUBLISHER] Endpoint %d for UserID %s: %s\n", i, userID, uuid.UUID(ep.ID()).String())
			_ = ep.SendTextDelta(sessionID, seq, text)
			// todo: emit error for text misses
		}
		return nil
	}
	// emit text broadcast failed event
	fmt.Printf("[PUBLISHER] FAILED: No endpoints found for UserID: %s\n", userID)
	return fmt.Errorf("couldn't broadcast text")
}

func (p *Publisher) SendAudioFrame(
	ctx context.Context,
	userID uuid.UUID,
	sessionID uuid.UUID,
	seq int,
	frame []byte,
) error {
	ep, ok := p.reg.SelectEndpointWithMRU(userID)
	if !ok || !ep.IsAlive() || !ep.Caps().AudioSink {
		return fmt.Errorf("couldn't send audio frame")
	}
	return ep.SendAudioFrame(sessionID, seq, frame)
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
