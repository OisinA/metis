package provider

import (
	"context"
	"metis/pkg/service"
	"metis/pkg/state"
)

type Provider interface {
	CreateService(context.Context, service.Service) (state.ServiceState, error)
	StartService(context.Context, state.ServiceState) (state.ServiceState, error)
	StopService(context.Context, state.ServiceState) (state.ServiceState, error)
	DestroyService(context.Context, state.ServiceState) (state.ServiceState, error)
	ServiceHealth(ctx context.Context, srv state.ServiceState) (state.ServiceState, error)
	GetServiceAddress(ctx context.Context, srv state.ServiceState) (string, error)
}
