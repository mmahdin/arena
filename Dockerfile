FROM golang:1.23 AS build
ARG GOPROXY=https://proxy.golang.org,direct
WORKDIR /src/apps/api
COPY apps/api/go.mod apps/api/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY apps/api/ ./
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go mod verify && go test ./... && \
    CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/migrate ./cmd/migrate
FROM alpine:3.21
WORKDIR /app
RUN addgroup -S app && adduser -S -G app app
COPY --from=build /out/api /app/api
COPY --from=build /out/migrate /app/migrate
COPY apps/api/db/migrations /app/migrations
USER app
EXPOSE 8080
CMD ["/app/api"]
