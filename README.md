# Metis

Super simple orchestration for stateless HTTP containers. This is a personal project I developed in an attempt to understand how a container orchestrator like Nomad/k8s would work under-the-hood.

Metis uses Traefik for load balancing between services (configuration can be checked in the `docker-compose.yml`).

## Security

There is a HTTP header passed around containing a token. It was not built with security in mind - do not use somewhere where security is important.

## Examples

### `projects/nginx.json`
```
{
    "name": "webserver",
    "configuration": {
        "image": "nginx",
        "count": 2,
        "container_port": 80,
        "host": "webserver.localhost"
    }
}
```

### `nodes/node0.json`
```
{
    "address": "host.docker.internal",
    "labels": [
        "host-system"
    ],
    "api_port": 6060
}
```