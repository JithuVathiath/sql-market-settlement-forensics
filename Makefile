GO ?= go
DC ?= docker compose
DB_EXEC = $(DC) exec -T db psql -v ON_ERROR_STOP=1 -U settlement -d settlement

.PHONY: help format lint test coverage build fixture profile db-up db-down db-migrate db-load sql-test benchmark docker-build quality

help:
	@echo "make quality     Run the full local quality gate"
	@echo "make fixture     Normalize and profile the deterministic fixture"
	@echo "make db-up       Start PostgreSQL 18"
	@echo "make sql-test    Migrate, load, and validate the warehouse"
	@echo "make benchmark   Run the one-million-row index benchmark"

format:
	$(GO) fmt ./...

lint:
	test -z "$$($(GO) fmt ./...)"
	$(GO) vet ./...

test:
	$(GO) test -race ./...

coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out
	@coverage=$$($(GO) tool cover -func=coverage.out | awk '/^total:/ {gsub("%", "", $$3); print $$3}'); \
	  awk -v coverage="$$coverage" 'BEGIN { if (coverage < 80) exit 1 }'

build:
	mkdir -p bin
	$(GO) build -trimpath -o bin/settlementctl ./cmd/settlementctl

fixture: build
	./bin/settlementctl normalize --input data/fixtures/sec_ftd_sample.txt --output data/processed/ftd.csv --rejects data/rejects/ftd-rejects.csv --receipt data/processed/normalization-receipt.json

profile: fixture
	./bin/settlementctl profile --input data/processed/ftd.csv --json reports/generated/source-profile.json --markdown reports/generated/source-profile.md

db-up:
	$(DC) up -d db

db-down:
	$(DC) down

db-migrate:
	$(DB_EXEC) -f sql/migrations/001_warehouse.sql
	$(DB_EXEC) -f sql/migrations/002_marts.sql

db-load: fixture
	$(DB_EXEC) -f sql/load_canonical.sql

sql-test: db-migrate db-load
	$(DB_EXEC) -f sql/quality/assertions.sql
	@for query in sql/analytics/*.sql; do $(DB_EXEC) -f "$$query" >/dev/null; done

benchmark:
	$(DB_EXEC) -v scale=1000000 -f benchmarks/setup.sql
	$(DB_EXEC) -f benchmarks/run.sql | tee benchmarks/results/latest.txt

docker-build:
	docker build --tag settlementctl:local .

quality: lint test coverage profile
