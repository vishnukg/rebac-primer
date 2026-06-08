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

## OpenFGA

OpenFGA runs as its own container:

```bash
make openfga/up
make openfga/seed
make server-openfga
```

The local OpenFGA setup uses an in-memory datastore, so restart means reseed.
