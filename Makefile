.PHONY: install build test coverage check openfga-up openfga-down

install:
	npm install

build:
	npm run build

test:
	npm test

coverage:
	npm run coverage

check:
	npm run check

openfga-up:
	docker compose -f deployments/docker-compose.yml up -d

openfga-down:
	docker compose -f deployments/docker-compose.yml down
