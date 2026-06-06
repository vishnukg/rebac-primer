# Docker Compose local services

Compose is a local orchestration file. It says which services exist, how to
build them, which ports they publish, and which environment variables they use.

Open `deployments/docker-compose.yml`.

## Scene

You want a one-command local environment. Not because one command is glamorous,
but because boring startup is what lets you focus on the authorization model.

## Services in this repo

```text
ts-authz      -> TypeScript AuthZ service (port 4100)
ts-documents  -> TypeScript Documents service (port 4000)
go-app        -> Go ReBAC server (port 4001)
openfga       -> local OpenFGA server
```

The two TypeScript services share the `ts-app` profile, so `--profile ts-app`
starts both.

The app services are behind profiles so you can choose whether to run either app
in Docker or directly on your host.

## Recommended local workflows

### Workflow A: everything through Make

This is the default workflow for this repo.

Build and test without local Node or Go:

```bash
make ts/check
make go/check
```

Run an app container:

```bash
make ts/server
make go/server
```

Run the TypeScript terminal client in another terminal:

```bash
make ts/client
```

### Workflow B: OpenFGA only

Start OpenFGA without the app container:

```bash
make openfga/up
```

Stop it:

```bash
make openfga/down
```

### Workflow C: raw Compose

You can still use Compose directly:

```bash
docker compose -f deployments/docker-compose.yml --profile ts-app up --build
docker compose -f deployments/docker-compose.yml --profile go-app up --build
```

But prefer `make` for day-to-day work so local and CI command shapes stay
consistent.

## What the current apps use

The current HTTP servers use in-memory graph authorizers so the
client/server demos work without a live OpenFGA store.

That is deliberate for the primer:

- you can learn client/server ReBAC first
- you can start OpenFGA separately
- later you can swap the graph authorizer for the OpenFGA adapter

The interface boundary is already there:

```ts
interface AuthzClient {
  check(request: CheckRequest): Promise<CheckResult>;
}
```

## Useful Compose commands

Prefer the Make targets:

```bash
make compose/config
make ts/server
make ts/server-down
make go/server
make go/server-down
```

Raw Compose equivalents:

Start services:

```bash
docker compose -f deployments/docker-compose.yml up
```

Start with an app profile:

```bash
docker compose -f deployments/docker-compose.yml --profile ts-app up
docker compose -f deployments/docker-compose.yml --profile go-app up
```

Rebuild:

```bash
docker compose -f deployments/docker-compose.yml --profile ts-app up --build
docker compose -f deployments/docker-compose.yml --profile go-app up --build
```

Stop and remove containers:

```bash
docker compose -f deployments/docker-compose.yml down
```

View logs:

```bash
docker compose -f deployments/docker-compose.yml logs -f ts-documents
docker compose -f deployments/docker-compose.yml logs -f go-app
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
curl http://127.0.0.1:4001/health
```

should return:

```json
{
  "ok": true
}
```

## Checkpoint

Why do the app services use Compose profiles?

Good answer: so you can run only OpenFGA when developing the app on your host,
or include one app container when you want to test that container path.
