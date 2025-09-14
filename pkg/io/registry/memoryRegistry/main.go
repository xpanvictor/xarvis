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
					ue = append(ue, ep)
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
	m.mu.RLock()
	defer m.mu.RUnlock()

	var endpoints []device.Endpoint
	if userDevices, exists := m.dvMap[userID]; exists {
		for _, d := range userDevices {
			for _, ep := range d.Endpoints {
				endpoints = append(endpoints, ep)
			}
		}
	}
	return endpoints
}

// RemoveDevice implements registry.Registry.
func (m *mmrRegistry) RemoveDevice(userID uuid.UUID, deviceID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if user exists
	userMap, exists := m.dvMap[userID]
	if !exists {
		return fmt.Errorf("user %s not found", userID)
	}

	// Check if device exists
	device, exists := userMap[deviceID]
	if !exists {
		return fmt.Errorf("device %s not found for user %s", deviceID, userID)
	}

	// Close all endpoints for this device
	for _, endpoint := range device.Endpoints {
		if closer, ok := endpoint.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				// Log error but continue cleanup
				fmt.Printf("Error closing endpoint: %v\n", err)
			}
		}
	}

	// Remove device from user's device map
	delete(userMap, deviceID)

	// If user has no more devices, remove user from registry
	if len(userMap) == 0 {
		delete(m.dvMap, userID)
	}

	return nil
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
func (m *mmrRegistry) SelectEndpointWithMRU(userID uuid.UUID, reqCap *device.Capabilities) (device.Endpoint, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if userMap, exists := m.dvMap[userID]; exists {
		// Sort devices by most recently used (descending order)
		devices := slices.SortedFunc(maps.Values(userMap), func(a, b *device.Device) int {
			return b.LastActive.Compare(a.LastActive) // Note: b first for descending order
		})

		// Check each device starting with most recently used
		for _, d := range devices {
			if d != nil && len(d.Endpoints) > 0 {
				// Sort endpoints by most recently used (descending order)
				se := slices.SortedFunc(maps.Values(d.Endpoints), func(a, b device.Endpoint) int {
					return b.LastActive().Compare(a.LastActive()) // Note: b first for descending order
				})

				// If no capability requirements, return the most recent endpoint
				if reqCap == nil {
					return se[0], true
				}

				// Check each endpoint for capability match
				for _, e := range se {
					if matchesCapabilities(e.Caps(), reqCap) {
						return e, true
					}
				}
			}
		}
	}
	return nil, false
}

// matchesCapabilities checks if endpoint capabilities satisfy the requirements
func matchesCapabilities(epCaps device.Capabilities, reqCaps *device.Capabilities) bool {
	if reqCaps == nil {
		return true
	}

	// Check each required capability
	if reqCaps.AudioSink && !epCaps.AudioSink {
		return false
	}
	if reqCaps.TextSink && !epCaps.TextSink {
		return false
	}
	if reqCaps.AudioWrite && !epCaps.AudioWrite {
		return false
	}

	return true
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

// Helper functions to create capability requirements

// RequireAudioSink creates capability requirements for audio sink only
func RequireAudioSink() *device.Capabilities {
	return &device.Capabilities{
		AudioSink:  true,
		TextSink:   false,
		AudioWrite: false,
	}
}

// RequireTextSink creates capability requirements for text sink only
func RequireTextSink() *device.Capabilities {
	return &device.Capabilities{
		AudioSink:  false,
		TextSink:   true,
		AudioWrite: false,
	}
}

// RequireAudioWrite creates capability requirements for audio write only
func RequireAudioWrite() *device.Capabilities {
	return &device.Capabilities{
		AudioSink:  false,
		TextSink:   false,
		AudioWrite: true,
	}
}

// RequireAudioSinkAndTextSink creates capability requirements for both audio and text sink
func RequireAudioSinkAndTextSink() *device.Capabilities {
	return &device.Capabilities{
		AudioSink:  true,
		TextSink:   true,
		AudioWrite: false,
	}
}

// RequireAllCapabilities creates capability requirements for all capabilities
func RequireAllCapabilities() *device.Capabilities {
	return &device.Capabilities{
		AudioSink:  true,
		TextSink:   true,
		AudioWrite: true,
	}
}

func New() registry.DeviceRegistry {
	return &mmrRegistry{
		dvMap: make(map[uuid.UUID]map[uuid.UUID]*device.Device, 0),
	}
}
