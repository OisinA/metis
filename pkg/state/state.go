package state

import (
	"metis/pkg/service"
	"metis/pkg/status"
)

type ServiceState struct {
	Status      status.ServiceStatus
	Service     service.DockerService
	Name        string
	ID          string
	ExposedPort int32
	Node        string
}
