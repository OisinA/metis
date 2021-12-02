package node

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"metis/internal/payload"
	"metis/pkg/service"
	"metis/pkg/state"
	"net/http"

	"github.com/spf13/viper"
)

type Node struct {
	Address string   `json:"address"`
	ID      string   `json:"id"`
	Labels  []string `json:"labels"`
	APIPort int      `json:"api_port"`
	Healthy bool     `json:"healthy"`
}

func (n Node) CreateService(ctx context.Context, srv service.Service) (state.ServiceState, error) {
	pload := payload.CreateServicePayload{
		Service: srv.(service.DockerService),
	}

	marshal, err := json.Marshal(pload)
	if err != nil {
		return state.ServiceState{}, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/service", n.Address, n.APIPort), bytes.NewBuffer(marshal))
	if err != nil {
		return state.ServiceState{}, err
	}
	req.Header.Set("Token", viper.GetString("metis.secret"))
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return state.ServiceState{}, err
	}

	respPload := payload.CreateServiceResponsePayload{}
	err = json.NewDecoder(resp.Body).Decode(&respPload)
	if err != nil {
		return state.ServiceState{}, err
	}

	respPload.ServiceState.Node = n.ID

	return respPload.ServiceState, nil
}

func (n Node) ServiceHealth(ctx context.Context, srv state.ServiceState) (state.ServiceState, error) {
	pload := payload.ServiceHealthPayload{
		ServiceState: srv,
	}

	marshal, err := json.Marshal(pload)
	if err != nil {
		return srv, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/service/health", n.Address, n.APIPort), bytes.NewBuffer(marshal))
	if err != nil {
		return srv, err
	}
	req.Header.Set("Token", viper.GetString("metis.secret"))
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return srv, err
	}

	respPload := payload.ServiceHealthResponsePayload{}
	err = json.NewDecoder(resp.Body).Decode(&respPload)
	if err != nil {
		return srv, err
	}

	return respPload.ServiceState, nil
}

func (n Node) DestroyService(ctx context.Context, srv state.ServiceState) (state.ServiceState, error) {
	pload := payload.DestroyServicePayload{
		ServiceState: srv,
	}

	marshal, err := json.Marshal(pload)
	if err != nil {
		return state.ServiceState{}, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s:%d/service/destroy", n.Address, n.APIPort), bytes.NewBuffer(marshal))
	if err != nil {
		return state.ServiceState{}, err
	}
	req.Header.Set("Token", viper.GetString("metis.secret"))
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return state.ServiceState{}, err
	}

	respPload := payload.DestroyServiceResponsePayload{}
	err = json.NewDecoder(resp.Body).Decode(&respPload)
	if err != nil {
		return state.ServiceState{}, err
	}

	return respPload.ServiceState, nil
}
