SHELL := /bin/sh

COMPOSE ?= docker compose -f deployments/docker-compose.yml
TOOLS := $(COMPOSE) --profile tools run --rm tools
APP := $(COMPOSE) --profile app

.DEFAULT_GOAL := help

.PHONY: help deps build test test-watch coverage check shell clean \
        server server-build server-down client openfga-up openfga-down \
        compose-config docker-build docker-test

help:
	@printf '%s\n' 'TS ReBAC Primer targets'
	@printf '%s\n' ''
	@printf '%s\n' '3 Musketeers workflow: make -> docker compose -> containerized tools'
	@printf '%s\n' ''
	@printf '%s\n' 'Core:'
	@printf '%s\n' '  make deps          Install npm dependencies inside the tools container volume'
	@printf '%s\n' '  make build         Compile TypeScript inside Docker'
	@printf '%s\n' '  make test          Run Vitest inside Docker'
	@printf '%s\n' '  make coverage      Run coverage inside Docker'
	@printf '%s\n' '  make check         Build and test inside Docker'
	@printf '%s\n' '  make shell         Open a shell in the tools container'
	@printf '%s\n' ''
	@printf '%s\n' 'Services:'
	@printf '%s\n' '  make server        Run the app container on http://127.0.0.1:4000'
	@printf '%s\n' '  make client        Run the interactive client in Docker'
	@printf '%s\n' '  make openfga-up    Start local OpenFGA'
	@printf '%s\n' '  make openfga-down  Stop local OpenFGA'
	@printf '%s\n' ''
	@printf '%s\n' 'Maintenance:'
	@printf '%s\n' '  make clean         Remove containers, volumes, build output, and coverage'
	@printf '%s\n' '  make compose-config Validate Compose config'

deps:
	$(TOOLS) npm ci

build:
	$(TOOLS) npm run build

test:
	$(TOOLS) npm test

test-watch:
	$(TOOLS) npm run test:watch

coverage:
	$(TOOLS) npm run coverage

check:
	$(TOOLS) npm run check

shell:
	$(TOOLS) sh

server-build:
	$(APP) build app

server:
	$(APP) up --build app

server-down:
	$(APP) down

client:
	$(TOOLS) npm run client

openfga-up:
	$(COMPOSE) up -d openfga

openfga-down:
	$(COMPOSE) down

compose-config:
	$(COMPOSE) --profile app --profile tools config

docker-build: build

docker-test: test

clean:
	$(COMPOSE) --profile app --profile tools down --volumes --remove-orphans
	rm -rf dist coverage
