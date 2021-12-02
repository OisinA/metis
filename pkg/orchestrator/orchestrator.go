package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"metis/pkg/node"
	"metis/pkg/project"
	"metis/pkg/service"
	"metis/pkg/state"
	"metis/pkg/status"
	"metis/pkg/traefik"
	"net/http"
	"os"

	"github.com/Strum355/log"
	"github.com/spf13/viper"
)

var (
	ROUNDROBIN = 0
)

type Orchestrator struct {
	Projects        []project.Project
	ProjectServices map[string][]state.ServiceState
	Nodes           map[string]node.Node
}

func NewOrchestrator() Orchestrator {
	return Orchestrator{
		Projects:        make([]project.Project, 0),
		ProjectServices: make(map[string][]state.ServiceState),
		Nodes:           make(map[string]node.Node),
	}
}

func OrchestratorFromState(state []byte) (Orchestrator, error) {
	var o Orchestrator
	err := json.Unmarshal(state, &o)
	if err != nil {
		return o, err
	}
	return o, nil
}

func (o *Orchestrator) WriteState() error {
	err := os.MkdirAll(viper.GetString("metis.home"), os.ModePerm)
	if err != nil {
		return err
	}

	writeLocation := viper.GetString("metis.home") + "/state.json"
	log.WithFields(log.Fields{
		"location": writeLocation,
	}).Info("Writing state")

	marsh, err := json.Marshal(o)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(writeLocation, marsh, 0644)
}

func (o *Orchestrator) NodeHealthcheck() {
	for i := range o.Nodes {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/", o.Nodes[i].Address, o.Nodes[i].APIPort), nil)
		if err != nil {
			log.WithFields(log.Fields{
				"node":    o.Nodes[i].ID,
				"address": o.Nodes[i].APIPort,
				"error":   err.Error(),
			}).Info("Node not healthy")
			continue
		}
		req.Header.Set("Token", viper.GetString("metis.secret"))
		client := http.Client{}
		response, err := client.Do(req)
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
	o.ProjectServices[proj.Name] = make([]state.ServiceState, 0)

	o.Projects = append(o.Projects, proj)

	return nil
}

func (o *Orchestrator) DestroyProject(proj project.Project) error {
	log.WithFields(log.Fields{
		"name": proj.Name,
	}).Info("Destroying project")
	for _, srv := range o.ProjectServices[proj.Name] {
		_, err := o.destroyService(srv)
		if err != nil {
			return err
		}
	}

	return nil
}

func (o *Orchestrator) GetServices() []state.ServiceState {
	states := []state.ServiceState{}
	for _, i := range o.ProjectServices {
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
}

func (o *Orchestrator) destroyService(srv state.ServiceState) (state.ServiceState, error) {
	ctx := context.Background()
	log.WithFields(log.Fields{
		"name": srv.Service.Name(),
	}).Info("Destroying service")

	return o.Nodes[srv.Node].DestroyService(ctx, srv)
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
	for k := range o.ProjectServices {
		for y := range o.ProjectServices[k] {
			srv, err := o.Nodes[o.ProjectServices[k][y].Node].ServiceHealth(ctx, o.ProjectServices[k][y])
			if err != nil {
				log.WithError(err).Error("Could not update status of service")
				srv.Status = status.UNHEALTHY
				o.ProjectServices[k][y] = srv
				continue
			}
			o.ProjectServices[k][y] = srv
		}
	}

	for project, services := range o.ProjectServices {
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

			o.ProjectServices[proj.Name] = append(o.ProjectServices[proj.Name], state)
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

	err = o.WriteState()
	if err != nil {
		return err
	}

	return nil
}

func (o *Orchestrator) removeStopped() error {
	for project, services := range o.ProjectServices {
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

		o.ProjectServices[project] = kept
	}

	return nil
}

func (o *Orchestrator) stopUnhealthy() error {
	for project, services := range o.ProjectServices {
		kept := []state.ServiceState{}
		for _, service := range services {
			if service.Status == status.UNHEALTHY {
				log.WithFields(log.Fields{
					"id":   service.ID,
					"name": service.Name,
				}).Info("Service unhealthy, removing service")
				_, err := o.destroyService(service)
				if err != nil {
					return err
				}
				continue
			}
			kept = append(kept, service)
		}

		o.ProjectServices[project] = kept
	}

	return nil
}

func (o *Orchestrator) CountHealthy(project string) (int, error) {
	services, ok := o.ProjectServices[project]
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
		services := o.ProjectServices[project.Name]
		router := traefik.Router{
			Rule:    fmt.Sprintf("Host(`%s`)", project.Configuration.Host),
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
