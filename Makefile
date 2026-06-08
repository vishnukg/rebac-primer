SHELL := /bin/sh

# Disable Docker Compose's interactive navigation menu. It adds nothing for
# these targets and, on terminals its TUI library does not recognise, prints a
# noisy "could not start menu ... termbox: unsupported terminal" warning.
COMPOSE_MENU ?= false
export COMPOSE_MENU

COMPOSE ?= docker compose -f deployments/docker-compose.yml
GO_TOOLS := $(COMPOSE) --profile tools run --rm tools
APP      := $(COMPOSE) --profile app

.DEFAULT_GOAL := help

.PHONY: help build test vet lint check shell server server-down server-openfga \
        openfga/up openfga/down openfga/seed compose/config clean

help:
	@printf '%s\n' 'ReBAC Primer — Go implementation'
	@printf '%s\n' ''
	@printf '%s\n' '3 Musketeers workflow: make -> docker compose -> containerized tools'
	@printf '%s\n' ''
	@printf '%s\n' 'Go:'
	@printf '%s\n' '  make build          Compile Go packages'
	@printf '%s\n' '  make test           Run Go tests'
	@printf '%s\n' '  make vet            Run go vet'
	@printf '%s\n' '  make lint           Run staticcheck (go tool)'
	@printf '%s\n' '  make check          Vet, staticcheck, and test'
	@printf '%s\n' '  make shell          Open shell in the Go tools container'
	@printf '%s\n' '  make server         Run the Go app on http://127.0.0.1:4001'
	@printf '%s\n' ''
	@printf '%s\n' 'OpenFGA:'
	@printf '%s\n' '  make openfga/up     Start local OpenFGA'
	@printf '%s\n' '  make openfga/down   Stop local OpenFGA'
	@printf '%s\n' '  make openfga/seed   Create store, write model, seed tuples'
	@printf '%s\n' '  make server-openfga Run app with AUTHZ_BACKEND=openfga'
	@printf '%s\n' ''
	@printf '%s\n' 'Cleanup:'
	@printf '%s\n' '  make clean          Remove containers, volumes, and build output'

build:
	$(GO_TOOLS) go build ./...

test:
	$(GO_TOOLS) go test ./...

vet:
	$(GO_TOOLS) go vet ./...

# staticcheck is pinned as a module tool dependency (the `tool` directive in
# go.mod), so `go tool staticcheck` builds it from the module — no global install.
lint:
	$(GO_TOOLS) go tool staticcheck ./...

check:
	$(GO_TOOLS) sh -c 'go vet ./... && go tool staticcheck ./... && go test ./...'

shell:
	$(GO_TOOLS) sh

server:
	$(APP) up --build app

server-down:
	$(APP) down

server-openfga:
	@test -f deployments/openfga/.ids.env || { echo "Run 'make openfga/up && make openfga/seed' first."; exit 1; }
	set -a; . deployments/openfga/.ids.env; set +a; \
	AUTHZ_BACKEND=openfga OPENFGA_API_URL=http://openfga:8080 $(APP) up --build app

openfga/up:
	$(COMPOSE) up -d openfga

openfga/down:
	$(COMPOSE) down

# Create the store, write the model, and seed policy tuples into the running
# OpenFGA (needs the fga CLI + jq). Writes deployments/openfga/.ids.env.
openfga/seed:
	deployments/openfga/seed.sh

compose/config:
	$(COMPOSE) --profile app --profile tools config

clean:
	$(COMPOSE) --profile app --profile tools down --volumes --remove-orphans
	rm -rf bin
