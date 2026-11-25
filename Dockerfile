FROM golang:1.23 AS build
ARG GOPROXY=https://proxy.golang.org,direct
WORKDIR /src/apps/api
COPY apps/api/go.mod apps/api/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY apps/api/ ./
# Tests read fixtures relative to the repository root.
COPY knowledge/ /src/knowledge/
COPY config/ai-runtime.yaml /src/config/ai-runtime.yaml
COPY docs/ai-core/inspector-examples/ /src/docs/ai-core/inspector-examples/
COPY docs/quality/test-results/ai-core-eval.json /src/docs/quality/test-results/ai-core-eval.json
COPY contracts/openapi/openapi.yaml /src/contracts/openapi/openapi.yaml
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go mod verify && go test ./... && \
    CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 go build -trimpath -o /out/staffctl ./cmd/staffctl
FROM alpine:3.21
WORKDIR /app
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/api /app/api
COPY --from=build /out/migrate /app/migrate
COPY --from=build /out/staffctl /app/staffctl
COPY apps/api/db/migrations /app/migrations
USER app
EXPOSE 8080
CMD ["/app/api"]
