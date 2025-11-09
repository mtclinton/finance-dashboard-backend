# Finance Dashboard Backend

Go backend service for the finance dashboard. Handles transactions, categories, and analytics with PostgreSQL and Redis caching.

## Setup

Make sure you have Go 1.21+ installed, then:

```bash
go mod download
```

## Running

The app expects a PostgreSQL database and optionally Redis for caching. Set these env vars:

- `DATABASE_URL` - PostgreSQL connection string (defaults to `postgres://postgres:postgres@postgres:5432/finance?sslmode=disable`)
- `REDIS_URL` - Redis connection string (defaults to `redis:6379`)
- `PORT` - Server port (defaults to `8080`)

### Database Setup

Run the migration to create tables and seed initial data:
```bash
go run . -migrate
```

Or if you've built the binary:
```bash
./main -migrate
```

The app will also auto-create the schema on startup if it doesn't exist, but running the migrate command separately is useful for Kubernetes init containers.

### Start the Server

Run it:
```bash
go run .
```

## API Endpoints

- `GET /health` - Health check
- `GET /api/transactions` - List transactions (cached 60s)
- `POST /api/transactions` - Create transaction
- `DELETE /api/transactions/:id` - Delete transaction
- `GET /api/categories` - List categories
- `GET /api/analytics` - Get analytics (cached 5min)

## Docker

Build the image:
```bash
docker build -t finance-dashboard-backend .
```

The app will auto-create the database schema and seed default categories on first run. Redis is optional - the app will work without it, just without caching.

