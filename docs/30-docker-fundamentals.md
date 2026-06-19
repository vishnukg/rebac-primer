# Docker Fundamentals

Docker gives this repo a repeatable way to run Go tools, the Go server, and
OpenFGA without relying on local setup beyond Docker itself.

## Images

Open `Dockerfile`.

The Dockerfile has two useful stages:

```text
dev      -> Go toolchain for build/test/shell work
runtime  -> compiled server binary
```

The `tools` Compose service uses the `dev` stage. The `app` service uses the
runtime image.

The final image contains only the compiled binary and a non-root user. The Go
toolchain stays in the build stages.

## Commands

```bash
make test
make check
make server
```

These run through Docker Compose by default. The same Go commands also work
from the repository root if you have Go installed locally:

```bash
go test ./...
go vet ./...
go run ./cmd/server
```

`make server` stays attached to the server logs. Run curl commands from a second
terminal, and stop the stack with `make server-down`.

## OpenFGA

OpenFGA runs as its own container:

```bash
make openfga/up
make openfga/seed
make server-openfga
```

The local OpenFGA setup uses an in-memory datastore, so restart means reseed.

## Checkpoint

Why use a multi-stage build? It keeps compilers and source files out of the
runtime image while preserving a reproducible build environment.
