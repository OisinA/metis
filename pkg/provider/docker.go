package provider

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"metis/pkg/service"
	"metis/pkg/state"
	"metis/pkg/status"
	"time"

	"github.com/Strum355/log"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type DockerProvider struct {
	client *client.Client
	state  map[string]state.ServiceState
}

func NewDockerProvider() (DockerProvider, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return DockerProvider{}, err
	}
	return DockerProvider{client: cli, state: make(map[string]state.ServiceState)}, nil
}

func (d *DockerProvider) GetContainers() ([]string, error) {
	containers, err := d.client.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return []string{}, err
	}

	cts := []string{}
	for _, cont := range containers {
		cts = append(cts, cont.Names[0])
	}

	return cts, nil
}

func (d *DockerProvider) CreateService(ctx context.Context, srvc service.Service) (state.ServiceState, error) {
	switch srvc.(type) {
	case service.DockerService:
		break
	default:
		return state.ServiceState{}, errors.New("Cannot pass a non-docker service to docker provider")
	}

	srv := srvc.(service.DockerService)

	// out, err := d.client.ImagePull(ctx, srv.DockerImage, types.ImagePullOptions{})
	// if err != nil {
	// 	log.WithError(err).Error("Could not pull image for docker service")
	// 	return err
	// }
	// defer out.Close()

	name := fmt.Sprintf("%s-%s", srv.SrvName, uuid.NewString()[:8])
	rand.Seed(time.Now().Unix())
	port := rand.Int31n(2000) + 4000
	resp, err := d.client.ContainerCreate(ctx, &container.Config{
		Image:        srv.DockerImage,
		ExposedPorts: nat.PortSet{nat.Port(fmt.Sprintf("%d/tcp", srv.ContainerPort)): struct{}{}},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(fmt.Sprintf("%d/tcp", srv.ContainerPort)): []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", port),
				},
			},
		},
	}, nil, nil, name)
	if err != nil {
		panic(err)
	}

	state := state.ServiceState{
		Status:      status.CREATED,
		Service:     srv,
		Name:        name,
		ID:          resp.ID,
		ExposedPort: port,
	}
	d.state[srv.SrvName] = state

	return state, nil
}

func (d *DockerProvider) StartService(ctx context.Context, srv state.ServiceState) (state.ServiceState, error) {
	if err := d.client.ContainerStart(ctx, srv.ID, types.ContainerStartOptions{}); err != nil {
		return state.ServiceState{}, err
	}
	return srv, nil
}

func (d *DockerProvider) StopService(ctx context.Context, srv state.ServiceState) (state.ServiceState, error) {
	timeout := 20 * time.Second
	if err := d.client.ContainerStop(ctx, srv.ID, &timeout); err != nil {
		return state.ServiceState{}, err
	}
	srv.Status = status.STOPPED

	return srv, nil
}

func (d *DockerProvider) DestroyService(ctx context.Context, srv state.ServiceState) (state.ServiceState, error) {
	if err := d.client.ContainerRemove(ctx, srv.ID, types.ContainerRemoveOptions{}); err != nil {
		return state.ServiceState{}, err
	}
	srv.Status = status.STOPPED
	delete(d.state, srv.Service.Name())
	return srv, nil
}

func (d *DockerProvider) ServiceHealth(ctx context.Context, srv state.ServiceState) (state.ServiceState, error) {
	cnt_json, err := d.client.ContainerInspect(ctx, srv.ID)
	if err != nil {
		log.WithFields(log.Fields{
			"old_status": srv.Status,
			"new_status": status.STOPPED,
		}).Info("Service health updated")
		log.WithError(err).Error("Could not find container")
		srv.Status = status.STOPPED
		return srv, nil
	}

	if cnt_json.State.Running && srv.Status != status.RUNNING {
		log.WithFields(log.Fields{
			"old_status": srv.Status,
			"new_status": status.RUNNING,
		}).Info("Service health updated")
		srv.Status = status.RUNNING
	} else if !cnt_json.State.Running && srv.Status == status.RUNNING {
		log.WithFields(log.Fields{
			"old_status": srv.Status,
			"new_status": status.UNHEALTHY,
		}).Info("Service health updated")
		srv.Status = status.UNHEALTHY
	}

	return srv, nil
}

func (d *DockerProvider) GetServiceAddress(ctx context.Context, srv state.ServiceState) (string, error) {
	cnt_json, err := d.client.ContainerInspect(ctx, srv.ID)
	if err != nil {
		return "", err
	}

	return cnt_json.NetworkSettings.IPAddress, nil
}
