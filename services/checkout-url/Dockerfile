# syntax=docker/dockerfile:1.7

# ---- Builder: compile static binary
FROM golang:1.26-alpine AS builder
WORKDIR /build

# Cache deps first to leverage layer caching.
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY checkout ./checkout
COPY config ./config
COPY httpapi ./httpapi
COPY logger ./logger
COPY midtrans ./midtrans

# Build static binary: CGO off, strip symbols/debug for smaller artifact.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-s -w' \
    -o /out/checkout-url \
    ./cmd/server

# ---- Runtime: distroless static (no shell, no package manager)
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/checkout-url /checkout-url
EXPOSE 8080
ENV APP_ENV=production

# Distroless has no shell or curl/wget, so HEALTHCHECK is intentionally
# omitted — orchestrators (k8s livenessProbe / docker-compose) should hit
# GET /health themselves.

ENTRYPOINT ["/checkout-url"]
