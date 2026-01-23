# go_cmdb

CDN Control Panel - Configuration Management Database

## Project Structure

```
go_cmdb/
├── cmd/
│   └── cmdb/
│       └── main.go          # Application entry point
├── api/
│   └── v1/
│       └── router.go        # API v1 routes
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration loader
│   ├── db/
│   │   └── mysql.go         # MySQL connection
│   └── cache/
│       └── redis.go         # Redis connection
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## Requirements

- Go 1.21+
- MySQL 5.7+ / 8.0+
- Redis 6.0+

## Configuration

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Required environment variables:

- `MYSQL_DSN` - MySQL connection string (required)
- `REDIS_ADDR` - Redis address (default: localhost:6379)
- `REDIS_PASS` - Redis password (default: empty)
- `REDIS_DB` - Redis database number (default: 0)
- `HTTP_ADDR` - HTTP server address (default: :8080)

## Installation

```bash
# Download dependencies
go mod download

# Build
go build -o bin/cmdb ./cmd/cmdb

# Run
./bin/cmdb
```

## Development

```bash
# Run directly
go run cmd/cmdb/main.go

# Run tests
go test ./...
```

## API Endpoints

### Health Check

```bash
GET /api/v1/ping
```

Response:
```json
{
  "code": 0,
  "message": "pong"
}
```

## Rollback Strategy

The application follows a **fail-fast** strategy:

- If MySQL connection fails → application exits immediately with error code 1
- If Redis connection fails → application exits immediately with error code 1
- If configuration loading fails → application exits immediately with error code 1

This ensures the application never runs in a partially initialized state.
