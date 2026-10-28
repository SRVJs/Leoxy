# Leoxy

A lightweight reverse proxy built in Go.

## Description

LX-2 routes HTTP requests to configurable upstream servers based on path prefixes. It supports HTTP/1.1, HTTP/2 upstreams, WebSocket upgrades, GraphQL over HTTP, and raw TCP/UDP traffic through dedicated listeners.

## Deployment architecture

LX-2 is designed to run behind Nginx for TLS termination. Do not expose it directly to the public internet.

```text
Client -> Nginx -> LX-2 -> Application servers
```

## Quick Start

### Using Docker

```bash
docker pull mohamed0699/leoxy:3.8.0
docker run --rm -p 4444:4444 -v "${PWD}/leoxy:/app/leoxy" mohamed0699/leoxy:latest
```

### Using Docker Compose

```yaml
services:
  leoxy:
    image: mohamed0699/leoxy:latest
    ports:
      - "4444:4444"
    volumes:
      - ./leoxy:/app/leoxy
    restart: unless-stopped
```

```bash
docker compose up -d
```

### Building from source

```bash
go build -o leoxy ./app/cli
./leoxy
```

## Configuration

LX-2 reads configuration from `leoxy/config.yaml`. On first start, a default config is created.

### Basic example

```yaml
server:
  port: "4444"

upstream:
  - name: "Backend"
    path: /api
    servers:
      - "http://localhost:7000"
      - "http://localhost:7001"

  - name: "Frontend"
    path: /
    server_url: "http://localhost:3000"
```

### TCP/UDP example

```yaml
upstream:
  - name: "GameTCP"
    protocol: tcp
    listen: ":7001"
    server_url: "127.0.0.1:7000"

  - name: "MetricsUDP"
    protocol: udp
    listen: ":7002"
    server_url: "127.0.0.1:7003"
```

## Features

- Reverse proxy with path-based routing
- Round-robin load balancing with failover
- Health checks (`/health/live`, `/health/ready`)
- Ping endpoint (`/ping`)
- Request/response metrics (`/metrics`)
- Rate limiting (per-IP, per-route, global)
- API key and JWT authentication
- mTLS support
- Request body size limits
- CIDR allow/deny lists
- WebSocket and GraphQL support
- TCP and UDP proxying

## CLI

```bash
leoxy             # Run in foreground
leoxy start       # Start in background
leoxy stop        # Stop background process
leoxy version     # Print version
```

## Security

- Rate limits are enforced per-minute with burst capacity
- Redis can be used for distributed global rate limiting
- API keys via `X-API-Key` header or Bearer token
- JWT authentication with HS256 (requires `exp`)
- mTLS support behind TLS terminator
- CIDR-based access control
- Request body limits

Set secrets via environment variables: `API_KEYS`, `JWT_SECRET`, `JWT_ISSUER`, `JWT_AUDIENCE`, `REDIS_PASSWORD`. Override port with `PORT`, rate limit with `RATE_LIMIT`.

## Endpoints

- `GET /ping` - Returns `PONG`
- `GET /health/live` - Liveness check
- `GET /health/ready` - Readiness check (checks HTTP upstreams)
- `GET /metrics` - Request metrics
- `GET /leoxy` - Project page

For more details, see [CLI documentation](docs/cli.md) and [configuration examples](example/).
