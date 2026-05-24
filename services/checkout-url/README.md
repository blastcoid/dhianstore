# checkout-url

Service Go + Fiber v3 yang nerima redirect dari Facebook/Instagram Shops checkout, bikin Midtrans Payment Link, lalu `302` redirect customer ke `payment_url`.

Overview repo + workflow: [CLAUDE.md di root](../../CLAUDE.md).

## Setup

```bash
cp .env.example .env
# isi MIDTRANS_SERVER_KEY / CLIENT_KEY / MERCHANT_ID dari sandbox dashboard
go run ./cmd/server
```

## Endpoints

- `GET /health` — liveness check
- `GET /checkout?products=<id:qty,id:qty>&coupon=&cart_origin=&fbclid=` — bikin payment link dan redirect 302

## Test

```bash
go test -race ./...
go test -race -coverprofile=coverage.out -coverpkg=./... ./...
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

Image: multi-stage `golang:1.26-alpine` → `gcr.io/distroless/static-debian12:nonroot`. Health check di-handle orchestrator (hit `GET /health`).
