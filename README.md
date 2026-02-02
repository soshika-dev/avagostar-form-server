# Smart Payment Registration System Backend (Flask)

Production-ready backend for the Smart Payment Registration System built with Flask and SQLAlchemy, using SQLite by default (PostgreSQL supported via configuration).

## Features
- Flask REST API with JWT authentication
- SQLite (default) or PostgreSQL via SQLAlchemy ORM
- Password hashing with bcrypt
- Rate limiting on auth endpoints
- Request ID header support
- CORS support for the Vue dev server
- Dashboard-ready filtering, sorting, pagination, and summary stats

## Prerequisites
- Python 3.10+

## Quick Start

```bash
cp .env.example .env
```

```bash
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```

```bash
python app.py
```

SQLite is the default database and stores data in `avagostar.db` in the project root. To use PostgreSQL instead, set `DATABASE_URL` to a PostgreSQL connection string.

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
| `DATABASE_URL` | Database connection string | `sqlite:///./avagostar.db` |
| `JWT_SECRET` | Secret for signing JWTs | `change-me` |
| `JWT_EXPIRES_IN` | JWT duration | `1h` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated list | `http://localhost:5173` |
| `RATE_LIMIT_PER_MIN` | Rate limit per IP for auth endpoints | `30` |
| `REQUEST_TIMEOUT` | DB timeout | `5s` |
