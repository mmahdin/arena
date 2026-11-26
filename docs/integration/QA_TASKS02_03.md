# Repeat local Tasks 02 v2 + 03 QA

Use synthetic `0900000XXXX` identifiers only. Keep all services on loopback and preserve the existing backend `.env` and DB volume.

From `wellbeing-server`:

```sh
docker compose build --build-arg GOPROXY=https://goproxy.io,direct
docker compose up --no-build -d
make acceptance-backend
```

From `ai-therapist-frontend`, first terminal:

```sh
npm ci
npm run dev
```

Second terminal:

```sh
npm test
npm run check:foundation
npm run build
BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:session
BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:chat
BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:qa
```

`BROWSER_BIN` is optional when Playwright's own browser is installed. The normal three E2E scripts do not intercept APIs. QA creates fresh fictional accounts; they remain in the local DB, while its conversations are deleted. Screenshots contain only synthetic data, without cookies or storage-state exports, under ignored `test-results/hamrah-qa/`.

For the separate controlled error suite, start an additional real Go API from `wellbeing-server` using the same existing `.env` / DB / secret:

```sh
HTTP_ADDR=127.0.0.1:8081 LLM_PROVIDER=openai-compatible MODEL_API_BASE_URL=https://127.0.0.1:1 MODEL_NAME=synthetic-failure-fixture MODEL_API_KEY=synthetic-local-fixture python3 scripts/run_local.py go run ./cmd/api
```

Then from the frontend:

```sh
BROWSER_BIN=/usr/bin/google-chrome npm run test:e2e:faults
```

Stop only this auxiliary process afterward. It calls an intentionally unreachable loopback provider, never an external LLM. The fault test forwards a request to the real failing API or drops a real committed response; no fabricated API success/reply is supplied. It also exercises real deleted-thread 404, revoked-session 401 and actual offline login recovery.

`check:source-baseline` compares bytes to the original designer ZIP. Four files already differed in the integration baseline, so its failure is expected and reported, not suppressed. `check:foundation` checks current required structure. Physical mobile keyboard/device testing and original-ZIP pixel comparison remain separate manual checks. See `VALIDATION_REPORT.md` for evidence and limits.
