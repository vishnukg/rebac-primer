# Docker Networking

Docker Compose creates a private network for the services in
`deployments/docker-compose.yml`.

## Host Ports

The Go app publishes:

```text
host 4001 -> container 4001
```

OpenFGA publishes:

```text
host 8080 -> OpenFGA HTTP API
host 8081 -> OpenFGA gRPC API
host 3000 -> OpenFGA playground
```

## Service Names

Inside Compose, services reach each other by service name. The Go app talks to
OpenFGA at:

```text
http://openfga:8080
```

From your host machine, use:

```text
http://127.0.0.1:8080
```

That is why `make server-openfga` sets `OPENFGA_API_URL=http://openfga:8080`
for the container.

## Quick Checks

```bash
make openfga/up
docker compose -f deployments/docker-compose.yml ps
curl http://127.0.0.1:8080/healthz
make openfga/seed
make server-openfga
curl http://127.0.0.1:4001/health
```

Run the final two commands in separate terminals because the server command
remains attached to logs.

## Checkpoint

Why does `localhost:8080` work from your laptop but not from the app container?
Inside a container, `localhost` means that same container. Compose DNS resolves
the peer service as `openfga`.
