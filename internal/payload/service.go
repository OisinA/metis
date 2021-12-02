package payload

import (
	"metis/pkg/service"
	"metis/pkg/state"
)

type CreateServicePayload struct {
	Service service.DockerService `json:"service"`
}

type CreateServiceResponsePayload struct {
	ServiceState state.ServiceState `json:"state"`
}

type ServiceHealthPayload struct {
	ServiceState state.ServiceState `json:"service"`
}

type ServiceHealthResponsePayload struct {
	ServiceState state.ServiceState `json:"state"`
}

type DestroyServicePayload struct {
	ServiceState state.ServiceState `json:"service"`
}

type DestroyServiceResponsePayload struct {
	ServiceState state.ServiceState `json:"state"`
}
