package service

import "metis/pkg/status"

type DockerService struct {
	SrvName       string               `json:"name"`
	DockerImage   string               `json:"docker_image"`
	DesiredStatus status.ServiceStatus `json:"desired_status"`
	ContainerPort int                  `json:"container_port"`
}

func (s DockerService) Name() string {
	return s.SrvName
}
