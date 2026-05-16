# Docker Compose local services

Compose is a local orchestration file. It says which services exist, how to
build them, which ports they publish, and which environment variables they use.

Open `deployments/docker-compose.yml`.

## Scene

You want a one-command local environment. Not because one command is glamorous,
but because boring startup is what lets you focus on the authorization model.

## Services in this repo

```text
app      -> TypeScript ReBAC HTTP server
openfga  -> local OpenFGA server
```

The app service is behind a profile so you can choose whether to run it in
Docker or directly with npm.

## Recommended local workflows

### Workflow A: everything through Make

This is the default workflow for this repo.

Build and test without local Node:

```bash
make check
```

Run the app container:

```bash
make server
```

Run the client in another terminal:

```bash
make client
```

### Workflow B: OpenFGA only

Start OpenFGA without the app container:

```bash
make openfga-up
```

Stop it:

```bash
make openfga-down
```

### Workflow C: raw Compose

You can still use Compose directly:

```bash
docker compose -f deployments/docker-compose.yml --profile app up --build
```

But prefer `make` for day-to-day work so local and CI command shapes stay
consistent.

## What the current app uses

The current HTTP server uses the in-memory `GraphAuthorizer` so the client/server
demo works without a live OpenFGA store.

That is deliberate for the primer:

- you can learn client/server ReBAC first
- you can start OpenFGA separately
- later you can swap `GraphAuthorizer` for `OpenFgaAuthorizer`

The interface boundary is already there:

```ts
interface Authorizer {
  check(request: CheckRequest): Promise<CheckResult>;
}
```

## Useful Compose commands

Prefer the Make targets:

```bash
make compose-config
make server
make server-down
```

Raw Compose equivalents:

Start services:

```bash
docker compose -f deployments/docker-compose.yml up
```

Start with app profile:

```bash
docker compose -f deployments/docker-compose.yml --profile app up
```

Rebuild:

```bash
docker compose -f deployments/docker-compose.yml --profile app up --build
```

Stop and remove containers:

```bash
docker compose -f deployments/docker-compose.yml down
```

View logs:

```bash
docker compose -f deployments/docker-compose.yml logs -f app
```

## Local service checklist

When a local service does not work, check:

1. Is the container running?
2. Is the port published?
3. Are you calling from host or from another container?
4. Is the app reading the expected environment variable?
5. Does `curl /health` work?

For this repo:

```bash
curl http://127.0.0.1:4000/health
```

should return:

```json
{
  "ok": true
}
```

## Checkpoint

Why does the app service use a Compose profile?

Good answer: so you can run only OpenFGA when developing the app on your host,
or include the app container when you want to test the container path.
