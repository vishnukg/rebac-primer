# TypeScript targets — a fragment included by the root Makefile. NOT standalone:
# run `make ts/test` (etc.) from the repo root, never `make -f typescript/ts.mk` here.
#
# It relies on variables defined in the root Makefile before the include:
#   TS_TOOLS  containerized Node toolchain runner (deps/build/test/shell/client)
#   TS_APP    docker compose invocation for the TS app profile
# Every recipe runs from the repo root, so relative paths (deployments/...) work.

.PHONY: ts/deps ts/build ts/test ts/test-watch ts/coverage ts/check ts/shell \
        ts/server ts/server-down ts/client ts/server-openfga ts/help

ts/deps:
	$(TS_TOOLS) npm ci

ts/build:
	$(TS_TOOLS) npm run build

ts/test:
	$(TS_TOOLS) npm test

ts/test-watch:
	$(TS_TOOLS) npm run test:watch

ts/coverage:
	$(TS_TOOLS) npm run coverage

ts/check:
	$(TS_TOOLS) npm run check

ts/shell:
	$(TS_TOOLS) sh

ts/server:
	$(TS_APP) up --build ts-authz ts-documents

ts/server-down:
	$(TS_APP) down

ts/client:
	$(TS_TOOLS) npm run client

# Run the TS app against the real OpenFGA backend. Requires `make openfga/up`
# and `make openfga/seed` first. The app containers reach OpenFGA by its compose
# service name (openfga:8080); the store/model IDs come from .ids.env.
ts/server-openfga:
	@test -f deployments/openfga/.ids.env || { echo "Run 'make openfga/up && make openfga/seed' first."; exit 1; }
	set -a; . deployments/openfga/.ids.env; set +a; \
	AUTHZ_BACKEND=openfga OPENFGA_API_URL=http://openfga:8080 $(TS_APP) up --build ts-authz ts-documents

ts/help:
	@printf '%s\n' 'TypeScript:'
	@printf '%s\n' '  make ts/deps        Install npm dependencies'
	@printf '%s\n' '  make ts/build       Compile TypeScript'
	@printf '%s\n' '  make ts/test        Run Vitest'
	@printf '%s\n' '  make ts/coverage    Run coverage'
	@printf '%s\n' '  make ts/check       Build and test'
	@printf '%s\n' '  make ts/shell       Open shell in the tools container'
	@printf '%s\n' '  make ts/server      Run the TS app on http://127.0.0.1:4000'
	@printf '%s\n' '  make ts/client      Run the interactive terminal client'
