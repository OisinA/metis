package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"metis/pkg/node"
	"metis/pkg/project"
	"metis/pkg/service"
	"metis/pkg/state"
	"metis/pkg/status"
	"metis/pkg/traefik"
	"net/http"

	"github.com/Strum355/log"
	"github.com/spf13/viper"
)

var (
	ROUNDROBIN = 0
)

type Orchestrator struct {
	Projects        []project.Project
	projectServices map[string][]state.ServiceState
	Nodes           map[string]node.Node
}

func NewOrchestrator() Orchestrator {
	return Orchestrator{
		Projects:        make([]project.Project, 0),
		projectServices: make(map[string][]state.ServiceState),
		Nodes:           make(map[string]node.Node),
	}
}

func (o *Orchestrator) NodeHealthcheck() {
	for i := range o.Nodes {
		response, err := http.Get(fmt.Sprintf("http://%s:%d/", o.Nodes[i].Address, o.Nodes[i].APIPort))
		if err != nil {
			nd := o.Nodes[i]
			nd.Healthy = false
			o.Nodes[i] = nd
			log.WithFields(log.Fields{
				"node":    o.Nodes[i].ID,
				"address": o.Nodes[i].APIPort,
				"error":   err.Error(),
			}).Info("Node not healthy")
			continue
		}

		if response.StatusCode != 200 {
			nd := o.Nodes[i]
			nd.Healthy = false
			o.Nodes[i] = nd
			log.WithFields(log.Fields{
				"node":          o.Nodes[i].ID,
				"address":       o.Nodes[i].APIPort,
				"response_code": response.StatusCode,
			}).Info("Node not healthy")
			continue
		}

		nd := o.Nodes[i]
		nd.Healthy = true
		o.Nodes[i] = nd
	}
}

func (o *Orchestrator) GetProjects() ([]project.Project, error) {
	return o.Projects, nil
}

func (o *Orchestrator) CreateProject(proj project.Project) error {
	log.WithFields(log.Fields{
		"name": proj.Name,
	}).Info("Creating project")
	o.projectServices[proj.Name] = make([]state.ServiceState, 0)

	o.Projects = append(o.Projects, proj)

	return nil
}

func (o *Orchestrator) DestroyProject(proj project.Project) error {
	log.WithFields(log.Fields{
		"name": proj.Name,
	}).Info("Destroying project")
	for _, srv := range o.projectServices[proj.Name] {
		// _, err := o.stopService(srv)
		// if err != nil {
		// 	return err
		// }
		_, err := o.destroyService(srv)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Orchestrator) GetServices() []state.ServiceState {
	states := []state.ServiceState{}
	for _, i := range o.projectServices {
		states = append(states, i...)
	}
	return states
}

func (o *Orchestrator) createService(srv service.Service) (state.ServiceState, error) {
	ctx := context.Background()
	log.WithFields(log.Fields{
		"name": srv.Name(),
	}).Info("Creating service")

	nodeIndex := ROUNDROBIN + 1
	if nodeIndex >= len(o.Nodes) {
		nodeIndex = 0
	}
	ROUNDROBIN = nodeIndex

	return o.Nodes[fmt.Sprintf("node-%d", nodeIndex)].CreateService(ctx, srv)

	// return o.jobProvider.CreateService(ctx, srv)
}

// func (o *Orchestrator) startService(srv state.ServiceState) (state.ServiceState, error) {
// 	ctx := context.Background()
// 	log.WithFields(log.Fields{
// 		"name": srv.Service.Name(),
// 	}).Info("Starting service")
// 	return o.jobProvider.StartService(ctx, srv)
// }

// func (o *Orchestrator) stopService(srv state.ServiceState) (state.ServiceState, error) {
// 	ctx := context.Background()
// 	log.WithFields(log.Fields{
// 		"name": srv.Service.Name(),
// 	}).Info("Stopping service")
// 	return o.jobProvider.StopService(ctx, srv)
// }

func (o *Orchestrator) destroyService(srv state.ServiceState) (state.ServiceState, error) {
	ctx := context.Background()
	log.WithFields(log.Fields{
		"name": srv.Service.Name(),
	}).Info("Destroying service")

	return o.Nodes[srv.Node].DestroyService(ctx, srv)
	// return o.jobProvider.DestroyService(ctx, srv)
}

func (o *Orchestrator) GetProject(name string) (project.Project, error) {
	for _, proj := range o.Projects {
		if proj.Name == name {
			return proj, nil
		}
	}

	return project.Project{}, errors.New("project not found")
}

func (o *Orchestrator) Update() error {
	ctx := context.Background()
	for k := range o.projectServices {
		for y := range o.projectServices[k] {
			// srv, err := o.jobProvider.ServiceHealth(ctx, o.projectServices[k][y])
			srv, err := o.Nodes[o.projectServices[k][y].Node].ServiceHealth(ctx, o.projectServices[k][y])
			if err != nil {
				log.WithError(err).Error("Could not update status of service")
				srv.Status = status.UNHEALTHY
				o.projectServices[k][y] = srv
				continue
			}
			o.projectServices[k][y] = srv
		}
	}

	for project, services := range o.projectServices {
		healthy := 0
		for _, service := range services {
			if service.Status == status.RUNNING || service.Status == status.CREATED {
				healthy++
			}
		}

		proj, err := o.GetProject(project)
		if err != nil {
			panic(err)
		}

		if healthy < proj.Configuration.Count {
			state, err := o.createService(service.DockerService{
				SrvName:       proj.Name,
				DockerImage:   proj.Configuration.ImageName,
				DesiredStatus: status.RUNNING,
				ContainerPort: proj.Configuration.ContainerPort,
			})
			if err != nil {
				return err
			}

			// state, err = o.startService(state)
			// if err != nil {
			// 	return err
			// }

			o.projectServices[proj.Name] = append(o.projectServices[proj.Name], state)
		}
	}

	err := o.removeStopped()
	if err != nil {
		return nil
	}
	err = o.stopUnhealthy()
	if err != nil {
		return nil
	}

	return nil
}

func (o *Orchestrator) removeStopped() error {
	for project, services := range o.projectServices {
		kept := []state.ServiceState{}
		for _, service := range services {
			if service.Status != status.STOPPED {
				kept = append(kept, service)
				continue
			}

			_, _ = o.destroyService(service)

			log.WithFields(log.Fields{
				"id":   service.ID,
				"name": service.Name,
			}).Info("Service stopped, removing service")
		}

		o.projectServices[project] = kept
	}

	return nil
}

func (o *Orchestrator) stopUnhealthy() error {
	for project, services := range o.projectServices {
		kept := []state.ServiceState{}
		for _, service := range services {
			if service.Status == status.UNHEALTHY {
				log.WithFields(log.Fields{
					"id":   service.ID,
					"name": service.Name,
				}).Info("Service unhealthy, removing service")
				// _, err := o.stopService(service)
				// if err != nil {
				// 	return err
				// }
				_, err := o.destroyService(service)
				if err != nil {
					return err
				}
				continue
			}
			kept = append(kept, service)
		}

		o.projectServices[project] = kept
	}

	return nil
}

func (o *Orchestrator) CountHealthy(project string) (int, error) {
	services, ok := o.projectServices[project]
	if !ok {
		return 0, errors.New("service not found")
	}
	healthy := 0
	for _, service := range services {
		if service.Status == status.RUNNING {
			healthy++
		}
	}

	return healthy, nil
}

func (o *Orchestrator) GetTraefikConfig() traefik.Configuration {
	ctx := context.Background()
	config := traefik.Configuration{HTTP: traefik.HttpConfig{}}
	config.HTTP.Routers = map[string]traefik.Router{}
	config.HTTP.Services = map[string]traefik.Service{}

	for _, project := range o.Projects {
		services := o.projectServices[project.Name]
		router := traefik.Router{
			Rule:    fmt.Sprintf("Host(`%s`)", viper.GetString("metis.controller.url")),
			Service: project.Name,
		}
		config.HTTP.Routers[project.Name+"-router"] = router

		urls := []traefik.URL{}
		for _, service := range services {
			service, err := o.Nodes[service.Node].ServiceHealth(ctx, service)
			if err != nil {
				log.WithError(err).Error("Could not fetch status for service")
				continue
			}
			if service.Status != status.RUNNING {
				continue
			}
			// url, err := o.jobProvider.GetServiceAddress(ctx, service)
			// if err != nil {
			// 	log.WithError(err).Error("Could not fetch URL for service")
			// 	continue
			// }
			url := o.Nodes[service.Node].Address
			urls = append(urls, traefik.URL{URL: fmt.Sprintf("http://%s:%d", url, service.ExposedPort)})
		}

		service := traefik.Service{
			LoadBalancer: traefik.LoadBalancer{
				Servers: urls,
			},
		}
		config.HTTP.Services[project.Name] = service
	}

	return config
}
