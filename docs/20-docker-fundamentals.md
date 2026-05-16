# Docker fundamentals

Docker lets you package an application with the runtime environment it needs.

For this repo, Docker has two jobs:

- run supporting services such as OpenFGA
- run the TypeScript ReBAC server in a repeatable local environment

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

Build the app image:

```bash
docker compose -f deployments/docker-compose.yml build app
```

Run the app profile:

```bash
docker compose -f deployments/docker-compose.yml --profile app up
```

Stop services:

```bash
docker compose -f deployments/docker-compose.yml down
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
