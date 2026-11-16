.PHONY: validate test integration-test acceptance-http acceptance-backend fmt build api migrate up down arch-check contract-check smoke
api:
	python3 scripts/run_local.py go run ./cmd/api
migrate:
	python3 scripts/run_local.py go run ./cmd/migrate
build:
	mkdir -p bin
	cd apps/api && go build -o ../../bin/api ./cmd/api && go build -o ../../bin/migrate ./cmd/migrate
fmt:
	cd apps/api && gofmt -w ./cmd ./internal
arch-check:
	python3 scripts/check_structure.py
contract-check:
	python3 scripts/check_contract.py
test:
	cd apps/api && go test ./...
integration-test:
	WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test ./tests/integration -v -count=1
acceptance-http:
	WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test ./tests/integration -run '^TestMVPHTTPAcceptance$$' -v -count=1
acceptance-backend: validate integration-test smoke
validate: arch-check contract-check test
up:
	docker compose up --build -d
down:
	docker compose down

smoke:
	python3 scripts/smoke_api.py
