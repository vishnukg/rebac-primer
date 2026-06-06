# Go targets — a fragment included by the root Makefile. NOT standalone:
# run `make go/test` (etc.) from the repo root, never `make -f go/go.mk` here.
#
# It relies on variables defined in the root Makefile before the include:
#   GO_TOOLS  containerized Go toolchain runner (build/test/vet/shell)
#   GO_APP    docker compose invocation for the Go app profile
# Every recipe runs from the repo root, so relative paths (deployments/...) work.

.PHONY: go/build go/test go/vet go/check go/shell \
        go/server go/server-down go/server-openfga go/help

go/build:
	$(GO_TOOLS) go build ./...

go/test:
	$(GO_TOOLS) go test ./...

go/vet:
	$(GO_TOOLS) go vet ./...

go/check:
	$(GO_TOOLS) sh -c 'go vet ./... && go test ./...'

go/shell:
	$(GO_TOOLS) sh

go/server:
	$(GO_APP) up --build go-app

go/server-down:
	$(GO_APP) down

# Run the Go app against the real OpenFGA backend. Requires `make openfga/up`
# and `make openfga/seed` first. The app container reaches OpenFGA by its compose
# service name (openfga:8080); the store/model IDs come from .ids.env.
go/server-openfga:
	@test -f deployments/openfga/.ids.env || { echo "Run 'make openfga/up && make openfga/seed' first."; exit 1; }
	set -a; . deployments/openfga/.ids.env; set +a; \
	AUTHZ_BACKEND=openfga OPENFGA_API_URL=http://openfga:8080 $(GO_APP) up --build go-app

go/help:
	@printf '%s\n' 'Go:'
	@printf '%s\n' '  make go/build       Compile Go binaries'
	@printf '%s\n' '  make go/test        Run Go tests'
	@printf '%s\n' '  make go/vet         Run go vet'
	@printf '%s\n' '  make go/check       Vet and test'
	@printf '%s\n' '  make go/shell       Open shell in the Go tools container'
	@printf '%s\n' '  make go/server      Run the Go app on http://127.0.0.1:4001'
