.PHONY: validate test test-scripts integration-test acceptance-http acceptance-backend fmt build api migrate env-sync up down arch-check contract-check smoke
api:
	AI_LOCAL_INSPECTOR=$${AI_LOCAL_INSPECTOR:-true} python3 scripts/run_local.py go run ./cmd/api
migrate:
	python3 scripts/run_local.py go run ./cmd/migrate
env-sync:
	python3 scripts/run_local.py --sync-env
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
test-scripts:
	python3 -m unittest discover -s scripts/tests -v
integration-test:
	WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test ./tests/integration -v -count=1
acceptance-http:
	WELLBEING_INTEGRATION=true python3 scripts/run_local.py go test ./tests/integration -run '^TestSQLServerFoundation$$' -v -count=1
acceptance-backend: validate integration-test
validate: arch-check contract-check test-scripts test
up: env-sync
	docker compose up --build -d
down:
	docker compose down

smoke:
	python3 scripts/auth/verify_tia_login_live.py --origin http://app.localhost:8080
