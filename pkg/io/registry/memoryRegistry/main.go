package memoryregistry

import (
	"fmt"
	"maps"
	"slices"
	"sync"
	"time"

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
func (m *mmrRegistry) FetchTextFanoutEndpoint(userID uuid.UUID) ([]device.Endpoint, bool) {
	if userDevices, exists := m.dvMap[userID]; exists {
		ue := make([]device.Endpoint, 0)
		for _, d := range userDevices {
			eps := slices.Collect(maps.Values(d.Endpoints))
			for _, ep := range eps {
				if ep.Caps().TextSink {
					eps = append(eps, ep)
				}
			}
			return ue, true
		}
	}
	return nil, false
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

func (m *mmrRegistry) TouchDevice(userID uuid.UUID, deviceID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if userMap, exists := m.dvMap[userID]; exists {
		if d, exists := userMap[deviceID]; exists {
			d.LastActive = time.Now()
			// touch all endpoints
			for _, e := range d.Endpoints {
				e.Touch()
			}
		}
	}
	return fmt.Errorf("can't find user")
}

// SelectEndpointWithMRU implements registry.Registry.
// todo: efficiency: rebalancing tree instead
func (m *mmrRegistry) SelectEndpointWithMRU(userID uuid.UUID) (device.Endpoint, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if userMap, exists := m.dvMap[userID]; exists {
		devices := slices.SortedFunc(maps.Values(userMap), func(a, b *device.Device) int { return a.LastActive.Compare(b.LastActive) })
		if len(devices) > 0 {
			if mrd := devices[0]; mrd != nil {
				se := slices.SortedFunc(maps.Values(mrd.Endpoints), func(a, b device.Endpoint) int { return a.LastActive().Compare(b.LastActive()) })
				if len(se) > 0 {
					return se[0], true
				}
			}
		}
	}
	return nil, false
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

func New() registry.DeviceRegistry {
	return &mmrRegistry{
		dvMap: make(map[uuid.UUID]map[uuid.UUID]*device.Device, 0),
	}
}
