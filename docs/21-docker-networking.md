# Docker networking

Local development usually needs multiple processes:

- the TypeScript app
- OpenFGA
- maybe a database later
- maybe a CLI on the host

Docker networking is how those processes find each other.

## Scene

The app says `ECONNREFUSED`. OpenFGA is running. The port looks right. The
problem is often not OpenFGA or TypeScript. It is where the request is coming
from: your host or another container.

## Host vs container networking

Inside a container, `localhost` means the container itself.

On your host machine, `localhost` means your laptop.

That difference matters.

If the app runs on your host and OpenFGA runs in Docker, the app can reach
OpenFGA through the published port:

```text
http://127.0.0.1:8080
```

If the app runs inside Docker Compose, it should reach OpenFGA by service name:

```text
http://openfga:8080
```

Compose creates DNS names for services.

## Service names

In `deployments/docker-compose.yml`:

```yaml
services:
  openfga:
    image: openfga/openfga:latest
```

Other Compose services can use:

```text
openfga
```

as a hostname.

That is why production-like config often differs from host-local config:

```text
host app -> http://127.0.0.1:8080
compose app -> http://openfga:8080
```

## Published ports

OpenFGA publishes:

```yaml
ports:
  - "8080:8080"
  - "8081:8081"
  - "3000:3000"
```

The left side is your host. The right side is the container.

Common OpenFGA ports:

- `8080`: API
- `8081`: playground or auxiliary service depending on image config
- `3000`: playground UI in common local setups

## Compose profiles

The app service uses a profile:

```yaml
profiles:
  - app
```

That means plain Compose can start only OpenFGA:

```bash
docker compose -f deployments/docker-compose.yml up openfga
```

And this starts the app too:

```bash
docker compose -f deployments/docker-compose.yml --profile app up
```

Profiles keep local infrastructure flexible.

## Debugging networking

Useful checks:

```bash
docker compose -f deployments/docker-compose.yml ps
```

```bash
curl http://127.0.0.1:4000/health
```

```bash
curl http://127.0.0.1:8080/healthz
```

If a container cannot reach another service, check:

1. Are they in the same Compose project?
2. Are you using service name from inside Docker?
3. Are you using published host port from outside Docker?
4. Is the service actually listening on that port?

## Rule of thumb

```text
From your laptop: use 127.0.0.1 plus published port.
From another Compose service: use service name plus container port.
```

## Checkpoint

If the app runs inside Compose, should it call OpenFGA at `127.0.0.1:8080` or
`openfga:8080`?

Good answer: `openfga:8080`, because `127.0.0.1` inside a container points back
to that same container.
