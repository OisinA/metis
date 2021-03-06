# Metis

Super simple orchestration for stateless HTTP containers. This is a personal project I developed in an attempt to understand how a container orchestrator like Nomad/k8s would work under-the-hood.

Metis uses Traefik for load balancing between services (configuration can be checked in the `docker-compose.yml`).

It is currently being used to host [https://oisinaylward.me](https://oisinaylward.me) across multiple nodes.

## Security

There is a HTTP header passed around containing a token. It was not built with security in mind - do not use somewhere where security is important.

## Configuration

Configuration is currently done entirely in the controller. It is entirely static - dynamic configuration is not yet possible. To change configuration, it requires bringing down the controller.

### Examples

#### `projects/nginx.json`
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

#### `nodes/node0.json`
```
{
    "address": "host.docker.internal",
    "labels": [
        "host-system"
    ],
    "api_port": 6060
}
```

## Deployment

Dockerfiles can be found in the docker directory for both the controller & agent. Check `docker-compose.yml` for a sample single-node deployment.