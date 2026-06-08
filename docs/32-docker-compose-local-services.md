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
