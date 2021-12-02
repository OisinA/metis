package status

type ServiceStatus string

const (
	CREATED   = "CREATED"
	RUNNING   = "RUNNING"
	STOPPED   = "STOPPED"
	UNHEALTHY = "UNHEALTHY"
)
