SHELL := /bin/sh

# Disable Docker Compose's interactive navigation menu. It adds nothing for
# these targets and, on terminals its TUI library does not recognise, prints a
# noisy "could not start menu ... termbox: unsupported terminal" warning.
COMPOSE_MENU ?= false
export COMPOSE_MENU

COMPOSE ?= docker compose -f deployments/docker-compose.yml

TS_TOOLS := $(COMPOSE) --profile ts-tools run --rm ts-tools
GO_TOOLS  := $(COMPOSE) --profile go-tools run --rm go-tools
TS_APP    := $(COMPOSE) --profile ts-app
GO_APP    := $(COMPOSE) --profile go-app

.DEFAULT_GOAL := help

.PHONY: help \
        ts-deps ts-build ts-test ts-test-watch ts-coverage ts-check ts-shell \
        ts-server ts-server-down ts-client ts-server-openfga \
        go-build go-test go-vet go-check go-shell \
        go-server go-server-down go-server-openfga \
        openfga-up openfga-down openfga-seed compose-config clean

help:
	@printf '%s\n' 'ReBAC Primer — TypeScript and Go implementations'
	@printf '%s\n' ''
	@printf '%s\n' '3 Musketeers workflow: make -> docker compose -> containerized tools'
	@printf '%s\n' ''
	@printf '%s\n' 'TypeScript:'
	@printf '%s\n' '  make ts-deps        Install npm dependencies'
	@printf '%s\n' '  make ts-build       Compile TypeScript'
	@printf '%s\n' '  make ts-test        Run Vitest'
	@printf '%s\n' '  make ts-coverage    Run coverage'
	@printf '%s\n' '  make ts-check       Build and test'
	@printf '%s\n' '  make ts-shell       Open shell in the tools container'
	@printf '%s\n' '  make ts-server      Run the TS app on http://127.0.0.1:4000'
	@printf '%s\n' '  make ts-client      Run the interactive terminal client'
	@printf '%s\n' ''
	@printf '%s\n' 'Go:'
	@printf '%s\n' '  make go-build       Compile Go binaries'
	@printf '%s\n' '  make go-test        Run Go tests'
	@printf '%s\n' '  make go-vet         Run go vet'
	@printf '%s\n' '  make go-check       Vet and test'
	@printf '%s\n' '  make go-shell       Open shell in the Go tools container'
	@printf '%s\n' '  make go-server      Run the Go app on http://127.0.0.1:4001'
	@printf '%s\n' ''
	@printf '%s\n' 'Shared:'
	@printf '%s\n' '  make openfga-up     Start local OpenFGA'
	@printf '%s\n' '  make openfga-down   Stop local OpenFGA'
	@printf '%s\n' '  make clean          Remove containers, volumes, and build output'
	@printf '%s\n' ''
	@printf '%s\n' 'Real OpenFGA backend (swap the from-scratch evaluator for OpenFGA):'
	@printf '%s\n' '  make openfga-up && make openfga-seed   Start + seed model/tuples (needs fga + jq)'
	@printf '%s\n' '  make go-server-openfga                 Go app, AUTHZ_BACKEND=openfga'
	@printf '%s\n' '  make ts-server-openfga                 TS app, AUTHZ_BACKEND=openfga'

# TypeScript targets
ts-deps:
	$(TS_TOOLS) npm ci

ts-build:
	$(TS_TOOLS) npm run build

ts-test:
	$(TS_TOOLS) npm test

ts-test-watch:
	$(TS_TOOLS) npm run test:watch

ts-coverage:
	$(TS_TOOLS) npm run coverage

ts-check:
	$(TS_TOOLS) npm run check

ts-shell:
	$(TS_TOOLS) sh

ts-server:
	$(TS_APP) up --build ts-authz ts-documents

ts-server-down:
	$(TS_APP) down

ts-client:
	$(TS_TOOLS) npm run client

# Go targets
go-build:
	$(GO_TOOLS) go build ./...

go-test:
	$(GO_TOOLS) go test ./...

go-vet:
	$(GO_TOOLS) go vet ./...

go-check:
	$(GO_TOOLS) sh -c 'go vet ./... && go test ./...'

go-shell:
	$(GO_TOOLS) sh

go-server:
	$(GO_APP) up --build go-app

go-server-down:
	$(GO_APP) down

# Shared
openfga-up:
	$(COMPOSE) up -d openfga

openfga-down:
	$(COMPOSE) down

# Create the store, write the model, and seed policy tuples into the running
# OpenFGA (needs the fga CLI + jq). Writes deployments/openfga/.ids.env.
openfga-seed:
	deployments/openfga/seed.sh

# Run the apps against the real OpenFGA backend. Requires `make openfga-up`
# and `make openfga-seed` first. The app containers reach OpenFGA by its compose
# service name (openfga:8080); the store/model IDs come from .ids.env.
go-server-openfga:
	@test -f deployments/openfga/.ids.env || { echo "Run 'make openfga-up && make openfga-seed' first."; exit 1; }
	set -a; . deployments/openfga/.ids.env; set +a; \
	AUTHZ_BACKEND=openfga OPENFGA_API_URL=http://openfga:8080 $(GO_APP) up --build go-app

ts-server-openfga:
	@test -f deployments/openfga/.ids.env || { echo "Run 'make openfga-up && make openfga-seed' first."; exit 1; }
	set -a; . deployments/openfga/.ids.env; set +a; \
	AUTHZ_BACKEND=openfga OPENFGA_API_URL=http://openfga:8080 $(TS_APP) up --build ts-authz ts-documents

compose-config:
	$(COMPOSE) --profile ts-app --profile ts-tools --profile go-app --profile go-tools config

clean:
	$(COMPOSE) --profile ts-app --profile ts-tools --profile go-app --profile go-tools down --volumes --remove-orphans
	rm -rf typescript/dist typescript/coverage go/bin
