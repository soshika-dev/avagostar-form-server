# Smart Payment Registration System Backend

Production-ready backend for the Smart Payment Registration System built with Go, Gin, and PostgreSQL.

## Features
- Gin REST API with JWT authentication
- PostgreSQL with SQL migrations (golang-migrate compatible)
- Password hashing with bcrypt
- Rate limiting on auth endpoints
- Request ID and structured logging
- CORS support for the Vue dev server
- Dashboard-ready filtering, sorting, pagination, and summary stats

## Prerequisites
- Docker + Docker Compose
- (Optional) `migrate` CLI or Docker profile for migrations

## Quick Start

```bash
cp .env.example .env
```

```bash
docker compose up --build
```

### Run Migrations
Using the built-in migrate container:

```bash
docker compose --profile tools run --rm migrate
```

### Seed Users
Demo users are seeded automatically on API startup:
- `admin` / `admin123`
- `user1` / `1111`
- `user2` / `2222`

## API Endpoints

### Health
```bash
curl http://localhost:8080/healthz
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'
```

### Forgot Password (dev returns code)
```bash
curl -X POST http://localhost:8080/api/v1/auth/forgot \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin"}'
```

### Reset Password
```bash
curl -X POST http://localhost:8080/api/v1/auth/reset \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","code":"123456","new_password":"NewPass123!"}'
```

### Create Transaction
```bash
curl -X POST http://localhost:8080/api/v1/transactions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer <TOKEN>' \
  -d '{
    "receiver_type":"individual",
    "receiver_name":"علی رضایی",
    "receiver_id":"optional",
    "payer_type":"legal",
    "payer_name":"شرکت ...",
    "payer_id":"optional",
    "payment_method":"cash",
    "currency":"IRR",
    "amount":1500000,
    "description":"شرح ...",
    "datetime_iso":"2026-01-28T10:12:00.000Z",
    "timezone":"Asia/Tehran"
  }'
```

### List Transactions
```bash
curl -X GET 'http://localhost:8080/api/v1/transactions?page=1&per_page=10&sort_by=date&sort_dir=desc' \
  -H 'Authorization: Bearer <TOKEN>'
```

### Summary
```bash
curl -X GET 'http://localhost:8080/api/v1/transactions/summary?currency=IRR' \
  -H 'Authorization: Bearer <TOKEN>'
```

## Environment Variables

| Variable | Description | Default |
| --- | --- | --- |
| `ENV` | `dev` or `prod` | `dev` |
| `HTTP_ADDR` | HTTP bind address | `:8080` |
| `DATABASE_URL` | Postgres connection string | `postgres://app:app@localhost:5432/avagostar?sslmode=disable` |
| `JWT_SECRET` | Secret for signing JWTs | `change-me` |
| `JWT_EXPIRES_IN` | JWT duration | `1h` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated list | `http://localhost:5173` |
| `RATE_LIMIT_PER_MIN` | Rate limit per IP for auth endpoints | `30` |
| `REQUEST_TIMEOUT` | DB timeout | `5s` |

## Development

Optional Makefile targets:

```bash
make run
make migrate-up
make migrate-down
```
