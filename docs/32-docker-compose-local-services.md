# Docker Compose Local Services

Open `deployments/docker-compose.yml`.

The local stack contains:

```text
tools   -> containerized Go toolchain
app     -> Go ReBAC server on port 4001
openfga -> local OpenFGA server
```

## Tooling

```bash
make test
make check
make shell
```

## Server

```bash
make server
```

In another terminal:

```bash
curl http://127.0.0.1:4001/health
```

## OpenFGA Backend

```bash
make openfga/up
make openfga/seed
make server-openfga
```

`openfga/seed` creates a store, uploads `deployments/openfga/model.fga`, writes
the demo policy tuples, and records the generated IDs in
`deployments/openfga/.ids.env`.

## Cleanup

```bash
make clean
```

## What to Notice

Profiles keep optional services out of commands that do not need them.
`make test` starts the tools profile; `make server` starts the app profile; OpenFGA can
run independently. This is orchestration for local learning, not a production
deployment topology.
