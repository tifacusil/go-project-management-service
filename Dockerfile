# syntax=docker/dockerfile:1

# ---- Build stage ----
FROM golang:1.25-alpine AS build

WORKDIR /src

# Cache deps layer: copy manifests first, download, then source.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Static, stripped binary. CGO off — pure-Go mysql driver, no libc needed.
# VERSION (hash del commit en CI) se inyecta en build-time vía linker flag -X:
# queda fijo en el binario, inmutable en runtime (no se puede pisar por env).
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux \
    go build -trimpath \
      -ldflags="-s -w -X github.com/tifacusil/go-project-management-service/internal/config.Version=${VERSION}" \
      -o /out/server ./cmd/server

# ---- Runtime stage ----
# Alpine en vez de distroless: el healthcheck del compose de infra usa `wget`,
# que distroless no incluye. Alpine trae wget (busybox) + CA certs + tzdata.
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 appuser

WORKDIR /app
COPY --from=build /out/server /app/server

# Puerto HTTP por defecto (el compose mapea 8090:8080).
EXPOSE 8080

USER appuser

ENTRYPOINT ["/app/server"]
