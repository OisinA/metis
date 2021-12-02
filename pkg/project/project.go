package project

type Project struct {
	Name          string               `json:"name"`
	Configuration ProjectConfiguration `json:"configuration"`
}

type ProjectConfiguration struct {
	ImageName     string `json:"image"`
	Count         int    `json:"count"`
	ContainerPort int    `json:"container_port"`
}
