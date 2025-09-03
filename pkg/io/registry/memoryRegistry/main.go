package memoryregistry

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/io/device"
	"github.com/xpanvictor/xarvis/pkg/io/registry"
)

type mmrRegistry struct {
	mu    sync.RWMutex
	dvMap map[uuid.UUID]map[uuid.UUID]*device.Device
}

// AttachEndpoint implements registry.Registry.
func (m *mmrRegistry) AttachEndpoint(userID uuid.UUID, deviceID uuid.UUID, ep device.Endpoint) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if userMap := m.dvMap[userID]; userMap != nil {
		if d := userMap[deviceID]; d != nil {
			if d.Endpoints == nil {
				d.Endpoints = make(map[device.EndpointID]device.Endpoint)
			}
			// can reinstantiate anyways
			d.Endpoints[ep.ID()] = ep
			return nil
		}
	}
	return fmt.Errorf("couldn't attach endpoint")
}

// DetachEndpoint implements registry.Registry.
func (m *mmrRegistry) DetachEndpoint(userID uuid.UUID, deviceID uuid.UUID, ep device.Endpoint) device.EndpointID {
	panic("unimplemented")
}

// FetchTextFanoutEndpoint implements registry.Registry.
func (m *mmrRegistry) FetchTextFanoutEndpoint(userID uuid.UUID) ([]*device.Endpoint, bool) {
	panic("unimplemented")
}

// ListUserDevices implements registry.Registry.
func (m *mmrRegistry) ListUserDevices(userID uuid.UUID) []device.Device {
	panic("unimplemented")
}

// ListUserEndpoints implements registry.Registry.
func (m *mmrRegistry) ListUserEndpoints(userID uuid.UUID) []device.Endpoint {
	panic("unimplemented")
}

// RemoveDevice implements registry.Registry.
func (m *mmrRegistry) RemoveDevice(userID uuid.UUID, deviceID uuid.UUID) error {
	panic("unimplemented")
}

// SelectEndpointWithMRU implements registry.Registry.
func (m *mmrRegistry) SelectEndpointWithMRU(userID uuid.UUID) (*device.Endpoint, bool) {
	panic("unimplemented")
}

// UpsertDevice implements registry.Registry.
func (m *mmrRegistry) UpsertDevice(userID uuid.UUID, d device.Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if userMap := m.dvMap[userID]; userMap == nil {
		m.dvMap[userID] = make(map[uuid.UUID]*device.Device)
	}
	// if dev := m.dvMap[userID][d.DeviceID]; dev != nil {
	// 	return
	// }
	m.dvMap[userID][d.DeviceID] = &d
	return nil
}

func New() registry.Registry {
	return &mmrRegistry{
		dvMap: make(map[uuid.UUID]map[uuid.UUID]*device.Device, 0),
	}
}
