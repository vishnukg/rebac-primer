# Docker fundamentals

Docker lets you package an application with the runtime environment it needs.

For this repo, Docker has two jobs:

- run supporting services such as OpenFGA
- run the TypeScript ReBAC server in a repeatable local environment
- run build/test tooling so local Node is optional

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

Open `deployments/Dockerfile`.

It has three stages:

```text
deps    -> install npm dependencies
build   -> compile TypeScript
runtime -> install production deps and run compiled JS
```

The runtime stage does not run `tsx`. It runs compiled JavaScript:

```dockerfile
CMD ["node", "dist/src/server.js"]
```

That is the production-shaped path. `tsx` is useful during development, but a
container should usually run built output.

## 3 Musketeers workflow

This repo follows the 3 Musketeers pattern:

```text
Makefile       -> developer task interface
Docker Compose -> service and tool orchestration
Docker         -> repeatable execution environment
```

The point is simple: a developer and CI should run the same command shape.

```bash
make test
```

does not call local `npm test`. It runs:

```text
docker compose run --rm tools npm test
```

That means you do not need local Node installed for normal project work. You need
Docker, Compose, and Make.

## Tool container

The `tools` Compose service uses the `deps` stage of the Dockerfile:

```yaml
tools:
  build:
    target: deps
  volumes:
    - ..:/workspace
    - node_modules:/workspace/node_modules
```

The source code is bind-mounted into `/workspace`. The `node_modules` directory
is stored in a Docker volume so dependency installs stay inside Docker and do
not pollute your host machine.

Use:

```bash
make deps
```

to refresh dependencies in that volume.

## Build context

When Compose builds the app, it uses the repo root as the build context:

```yaml
build:
  context: ..
  dockerfile: deployments/Dockerfile
```

The context is the directory Docker can read while building. Files outside the
context are invisible to the build.

## Layers and caching

The Dockerfile copies dependency manifests before source:

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

The app listens on port `4000` inside the container:

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

## Environment variables

The app reads:

```text
PORT
```

Compose sets:

```yaml
environment:
  PORT: "4000"
```

Keep configuration in environment variables when it changes per environment.
Keep code and images the same.

## Commands

Run the normal project lifecycle:

```bash
make deps
make build
make test
make coverage
make check
```

Open a shell in the tool container:

```bash
make shell
```

Build the app image:

```bash
make server-build
```

Run the app profile:

```bash
make server
```

Stop services:

```bash
make server-down
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
