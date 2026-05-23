# checkout-url

Service yang nerima redirect dari Facebook/Instagram Shops checkout, bikin
Midtrans Payment Link, lalu `302` redirect customer ke `payment_url`.
Implemented dengan Go + Fiber v3.

Untuk overview repo dan workflow, lihat [CLAUDE.md di root](../../CLAUDE.md).

## Setup

```bash
cp .env.example .env
# isi MIDTRANS_SERVER_KEY / CLIENT_KEY / MERCHANT_ID dari dashboard sandbox
go run ./cmd/server
```

## Endpoints

- `GET /health` — liveness check.
- `GET /checkout?products=<id:qty,id:qty>&coupon=&cart_origin=&fbclid=` — bikin payment link dan redirect 302.

## Test

```bash
go test ./...
go test -coverprofile=coverage.out -coverpkg=./internal/... ./...
go tool cover -func=coverage.out
```

## Lint

```bash
golangci-lint run ./...
```

## Docker

```bash
docker build -t checkout-url .
docker run --env-file .env -p 8080:8080 checkout-url
```

Image runtime pakai distroless (no shell, non-root). Health check tidak built-in
ke image — orchestrator (k8s livenessProbe / docker-compose healthcheck)
harus hit `GET /health` sendiri.
