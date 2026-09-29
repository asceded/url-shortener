# URL Shortener

A production-grade URL shortener built with Go, Gin, PostgreSQL, and Redis. Supports link creation, redirects, click analytics, and async event processing via a worker pool.

## Features

- Create short links from long URLs
- Redirect via HTTP 302
- Click analytics: total clicks, last click time
- Async click tracking with a worker pool and batch inserts
- Redis caching for fast redirects
- Graceful shutdown
- Dockerized environment (PostgreSQL + Redis)

## Tech Stack

- Language: Go 1.22
- HTTP: Gin
- Database: PostgreSQL 16 (pgx driver)
- Cache: Redis 7
- Containerization: Docker, Docker Compose
- Logging: log/slog (structured JSON)

## Architecture

Client
  |
  v
[Gin Handler] --> [Service] --> [Cache (Redis)]
                     |
                     v
              [Repository (Postgres)]
                     |
                     v
              [Worker Pool] --> [Batch INSERT]

The redirect path uses a cache-aside pattern: Redis is checked first, and on miss the link is fetched from PostgreSQL and cached. Click events are pushed to a buffered channel and flushed to the database in batches by a background worker.

## API

### Create a short link

POST /api/links
Content-Type: application/json

{
  "url": "https://example.com/some/long/path"
}

Response:

{
  "code": "aB3xK9z",
  "short_url": "http://localhost:8080/aB3xK9z",
  "original_url": "https://example.com/some/long/path"
}

### Redirect

GET /:code

Returns 302 Found with Location header pointing to the original URL.

### Get stats

GET /api/links/:code/stats

Response:

{
  "code": "aB3xK9z",
  "total_clicks": 42,
  "last_click": "2026-09-29T19:57:12Z"
}

### Delete a link

DELETE /api/links/:code

Returns 204 No Content.

## Getting Started

### Prerequisites

- Go 1.22+
- Docker and Docker Compose

### Run

1. Clone the repository:

git clone https://github.com/asceded/url-shortener.git
cd url-shortener

2. Create .env from the example:

cp .env.example .env

3. Start PostgreSQL and Redis:

docker-compose up -d

4. Run the application:

go run ./cmd/api

The server will start on http://localhost:8080.

## Project Structure

url-shortener/
  cmd/api/              # application entry point
  internal/
    config/             # configuration loading
    handler/            # HTTP handlers (Gin)
    model/              # domain models
    repository/         # PostgreSQL and Redis access
    service/            # business logic
    worker/             # async click processing
  migrations/           # SQL migrations
  docker-compose.yml
  Dockerfile
  README.md

## Design Decisions

- pgx over database/sql: native PostgreSQL protocol support, connection pooling, and batch queries.
- Cache-aside with Redis: reduces database load on the hot redirect path.
- Worker pool with batching: decouples redirect latency from write throughput. Clicks are buffered in a channel and flushed in batches of 100 or every 2 seconds.
- Structured logging with slog: JSON logs, ready for aggregation.
- Graceful shutdown: in-flight requests are drained and the final batch is flushed before exit.
- Base62-style code generation: crypto/rand for unpredictable, URL-safe codes.

## Possible Improvements

- gRPC API for internal service-to-service calls
- Rate limiting per IP
- Prometheus metrics and Grafana dashboards
- Custom short codes
- Link expiration with TTL
- Click deduplication

## License

MIT