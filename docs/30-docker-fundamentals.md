# Docker fundamentals

Docker lets you package an application with the runtime environment it needs.

For this repo, Docker has three jobs:

- run supporting services such as OpenFGA
- run the TypeScript and Go ReBAC servers in repeatable local environments
- run build/test tooling so local Node and Go installs are optional

## Scene

You want to run the same service tomorrow without remembering which Node version,
which command, or which OpenFGA ports were needed. Docker gives you a repeatable
box for the process.

## Image vs container

An image is a packaged filesystem and startup command.

A container is a running process created from an image.

```text
image      -> recipe and filesystem
container  -> running process from that recipe
```

You can create many containers from the same image.

## Dockerfile

Open `typescript/Dockerfile` and `go/Dockerfile`.

The TypeScript Dockerfile has three stages:

```text
deps    -> FROM node:22-slim;  install npm dependencies
build   -> FROM deps;          add src + test, compile TypeScript
runtime -> FROM node:22-slim;  install prod-only deps, copy /dist, run node
```

Two things to notice: `build` extends `deps` (so it inherits `node_modules`
without reinstalling), and `runtime` starts fresh from `node:22-slim` and
re-installs with `--omit=dev` so the final image carries no build tools or
test code.

The runtime stage does not run `tsx`. It runs compiled JavaScript:

```dockerfile
CMD ["node", "dist/src/server.js"]
```

That is the production-shaped path. `tsx` is useful during development, but a
container should usually run built output.

The Go Dockerfile follows the same idea with different tooling:

```text
dev     -> Go toolchain for build/test work
build   -> compile the server binary
runtime -> run the compiled binary in a small Alpine image
```

## 3 Musketeers workflow

This repo follows the 3 Musketeers pattern:

```text
Makefile       -> developer task interface
Docker Compose -> service and tool orchestration
Docker         -> repeatable execution environment
```

The point is simple: a developer and CI should run the same command shape.

```bash
make ts-test
make go-test
```

do not call local test tools directly. They run through Compose tool containers:

```text
docker compose run --rm ts-tools npm test
docker compose run --rm go-tools go test ./...
```

That means you do not need local Node or Go installed for normal project work.
You need Docker, Compose, and Make.

## Tool container

The `ts-tools` Compose service uses the `deps` stage of the TypeScript Dockerfile:

```yaml
ts-tools:
  build:
    context: ../typescript
    dockerfile: Dockerfile
    target: deps
  volumes:
    - ../typescript:/workspace
    - ts_node_modules:/workspace/node_modules
```

The source code is bind-mounted into `/workspace`. The `node_modules` directory
is stored in a Docker volume so dependency installs stay inside Docker and do
not pollute your host machine.

Use:

```bash
make ts-deps
```

to refresh dependencies in that volume.

The `go-tools` service uses the Go `dev` stage and a Go module cache volume:

```yaml
go-tools:
  build:
    context: ../go
    dockerfile: Dockerfile
    target: dev
  volumes:
    - ../go:/workspace
    - go_cache:/root/go/pkg/mod
```

## Build context

When Compose builds each app, it uses that language directory as the build
context:

```yaml
build:
  context: ../typescript
  dockerfile: Dockerfile
```

For Go:

```yaml
build:
  context: ../go
  dockerfile: Dockerfile
```

The context is the directory Docker can read while building. Files outside that
context are invisible to the build.

## Layers and caching

The TypeScript Dockerfile copies dependency manifests before source:

```dockerfile
COPY package.json package-lock.json ./
RUN npm ci
```

Then it copies source:

```dockerfile
COPY src ./src
```

This improves caching. If source changes but dependencies do not, Docker can
reuse the npm install layer.

## Ports

The TypeScript app listens on port `4000` inside the container:

```dockerfile
EXPOSE 4000
```

Compose publishes it to your host:

```yaml
ports:
  - "4000:4000"
```

Format:

```text
host_port:container_port
```

So `http://127.0.0.1:4000` on your machine reaches port `4000` in the
container.

The Go app uses the same pattern on port `4001`:

```yaml
ports:
  - "4001:4001"
```

## Environment variables

Both apps read:

```text
PORT
```

Compose sets `4000` for TypeScript and `4001` for Go:

```yaml
ts-app:
  environment:
    PORT: "4000"

go-app:
  environment:
    PORT: "4001"
```

Keep configuration in environment variables when it changes per environment.
Keep code and images the same.

## Commands

Run the normal project lifecycle:

```bash
make ts-deps
make ts-check
make go-check
```

Open a shell in the tool container:

```bash
make ts-shell
make go-shell
```

Run an app profile:

```bash
make ts-server
make go-server
```

Stop an app profile:

```bash
make ts-server-down
make go-server-down
```

## What to remember

Docker is not magic. It is process isolation plus filesystem packaging plus
network wiring.

The most useful mental model:

```text
Dockerfile says how to build one service.
Compose says how local services run together.
```

## Checkpoint

Explain the difference:

```text
image: packaged recipe
container: running process from that recipe
```
