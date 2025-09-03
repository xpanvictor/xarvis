package registry

import (
	"github.com/google/uuid"
	"github.com/xpanvictor/xarvis/pkg/io/device"
)

type Registry interface {
	// device lifecyle
	UpsertDevice(userID uuid.UUID, d device.Device) error
	RemoveDevice(userID uuid.UUID, deviceID uuid.UUID) error
	// endpoint lifecycle
	AttachEndpoint(userID uuid.UUID, deviceID uuid.UUID, ep device.Endpoint) error
	DetachEndpoint(userID uuid.UUID, deviceID uuid.UUID, ep device.Endpoint) device.EndpointID
	// queries
	ListUserDevices(userID uuid.UUID) []device.Device
	ListUserEndpoints(userID uuid.UUID) []device.Endpoint
	// selection
	SelectEndpointWithMRU(userID uuid.UUID) (*device.Endpoint, bool)
	FetchTextFanoutEndpoint(userID uuid.UUID) ([]*device.Endpoint, bool)
}
