SHELL := /bin/sh

# Disable Docker Compose's interactive navigation menu. It adds nothing for
# these targets and, on terminals its TUI library does not recognise, prints a
# noisy "could not start menu ... termbox: unsupported terminal" warning.
COMPOSE_MENU ?= false
export COMPOSE_MENU

COMPOSE ?= docker compose -f deployments/docker-compose.yml

# Containerized tool/app runners shared by the language-specific makefiles.
# Defined here (before the includes) so make/go.mk and make/ts.mk can use them.
TS_TOOLS := $(COMPOSE) --profile ts-tools run --rm ts-tools
GO_TOOLS  := $(COMPOSE) --profile go-tools run --rm go-tools
TS_APP    := $(COMPOSE) --profile ts-app
GO_APP    := $(COMPOSE) --profile go-app

.DEFAULT_GOAL := help

# Language-specific targets live in their own makefiles. This root ties them
# together and owns the shared (OpenFGA / cleanup) targets. They are namespaced
# with a slash — `make go/test`, `make ts/server`, etc. — and defined in
# make/go.mk and make/ts.mk. Everything still runs from the repo root.
include make/go.mk
include make/ts.mk

.PHONY: help openfga/up openfga/down openfga/seed compose/config clean

help:
	@printf '%s\n' 'ReBAC Primer — TypeScript and Go implementations'
	@printf '%s\n' ''
	@printf '%s\n' '3 Musketeers workflow: make -> docker compose -> containerized tools'
	@printf '%s\n' ''
	@$(MAKE) --no-print-directory ts/help
	@printf '%s\n' ''
	@$(MAKE) --no-print-directory go/help
	@printf '%s\n' ''
	@printf '%s\n' 'Shared:'
	@printf '%s\n' '  make openfga/up     Start local OpenFGA'
	@printf '%s\n' '  make openfga/down   Stop local OpenFGA'
	@printf '%s\n' '  make clean          Remove containers, volumes, and build output'
	@printf '%s\n' ''
	@printf '%s\n' 'Real OpenFGA backend (swap the from-scratch evaluator for OpenFGA):'
	@printf '%s\n' '  make openfga/up && make openfga/seed   Start + seed model/tuples (needs fga + jq)'
	@printf '%s\n' '  make go/server-openfga                 Go app, AUTHZ_BACKEND=openfga'
	@printf '%s\n' '  make ts/server-openfga                 TS app, AUTHZ_BACKEND=openfga'

# ── Shared targets ──────────────────────────────────────────────────────────

openfga/up:
	$(COMPOSE) up -d openfga

openfga/down:
	$(COMPOSE) down

# Create the store, write the model, and seed policy tuples into the running
# OpenFGA (needs the fga CLI + jq). Writes deployments/openfga/.ids.env.
openfga/seed:
	deployments/openfga/seed.sh

compose/config:
	$(COMPOSE) --profile ts-app --profile ts-tools --profile go-app --profile go-tools config

clean:
	$(COMPOSE) --profile ts-app --profile ts-tools --profile go-app --profile go-tools down --volumes --remove-orphans
	rm -rf typescript/dist typescript/coverage go/bin
