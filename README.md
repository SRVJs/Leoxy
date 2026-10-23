# Leoxy (LX-2)

LX-2 is a lightweight reverse proxy built in Go. It routes HTTP requests to configurable upstream servers based on path prefixes, and can also forward raw TCP and UDP traffic through dedicated listeners.

## Deployment architecture

LX-2 is designed to run behind Nginx:

```text
Client -> Nginx -> LX-2 -> Application servers
```

Nginx is responsible for public HTTPS termination and forwarding requests to LX-2 on a private network. LX-2 applies request-body and rate limits, then routes approved requests to the configured application servers.

Do not expose LX-2 directly to the public Internet when Nginx is the TLS terminator. Bind it to a private interface or firewall its port so only Nginx can reach it.

### Request-body limits

Set `server.security.max_body` for a global maximum request-body size in megabytes. Set `upstream[].body` or `upstream[].security.max_body` to apply a limit to one upstream. Requests above the configured limit are rejected with `413 Request Entity Too Large` before they reach the application server.

### Rate limits

LX-2 supports per-IP, per-route, and global rate limits. Configure them under `server.security` for proxy-wide limits and `upstream[].security` for route-specific limits. Rates are requests per minute and burst values define the initial request capacity.

When Nginx forwards traffic, ensure LX-2 receives client-IP headers only from that trusted Nginx instance. Do not allow clients to connect directly to LX-2 and submit forwarded-IP headers.


## Uses

Pull and run a published version from Docker Hub:

```bash
docker pull elrefai99/leoxy:3.8.0
docker run --rm -p 8080:8080 -v "${PWD}/leoxy:/app/leoxy" elrefai99/leoxy:latest
```

The container reads `leoxy/config.yaml`. On first start it creates a default config in the mounted `./leoxy` directory. Edit its upstream URLs so they are reachable from the container. For a backend in the same Compose project, use the backend service name as the hostname instead of `localhost`.

Example `compose.yaml`:

```yaml
services:
  leoxy:
    image: elrefai99/leoxy:latest
    ports:
      - "8080:8080"
    volumes:
      - ./leoxy:/app/leoxy
    restart: unless-stopped
```

Start the service and view its logs with:

```bash
docker compose up -d
docker compose logs -f leoxy
```

Change the image tag to `latest` to follow the latest published build. The version tag is read from `version.json` during the Docker workflow.


## Features

- **Reverse proxy** – forwards requests to multiple backend services using `httputil.ReverseProxy`.
- **Load balancing** – round-robin routing across multiple servers with connection-failure failover for bodyless or replayable requests.
- **Path-based routing** – each upstream is mapped to a configurable path prefix; the prefix is stripped before forwarding.
- **Health checks** – exposes liveness and ping endpoints for monitoring.
- **Request logging** – logs method, path, remote address, and latency for every request.
- **CLI** – start, stop, and run the service as a background process with `cobra`-based subcommands.
- **YAML configuration** – upstreams and server settings are defined in `leoxy/config.yaml` via `viper`.
- **Timeouts** – explicit read/write/idle timeouts and max header size for safety.

Leoxy supports HTTP/1.1 and upstream HTTP/2, WebSocket upgrades, and GraphQL over HTTP. TCP and UDP use separately configured raw listeners. HTTP/3 is not currently supported.

## Configuration

Create `leoxy/config.yaml`:

```yaml
server:
  port: "8081"

upstream:
  - name: "GameTCP"
    protocol: tcp
    listen: ":7001"
    server_url: "127.0.0.1:7000"

  - name: "MetricsUDP"
    protocol: udp
    listen: ":7002"
    server_url: "127.0.0.1:7003"

  - name: "Backend"
    path: /api
    ip: true
    servers:
      - "http://localhost:7000"
      - "http://localhost:7001"

  - name: "Frontend"
    path: /
    server_url: "http://localhost:3000"
```

| Key | Description |
|-----|-------------|
| `server.port` | Listen address (defaults to `:8080`; a bare number gets `:` prepended). |
| `upstream[].name` | Identifier for the upstream. |
| `upstream[].path` | Route prefix that triggers this upstream (defaults to `/`). |
| `upstream[].server_url` | Absolute URL of the backend; validated at startup. |
| `upstream[].servers` | List of backend URLs for round-robin load balancing. Failed connections are retried when the request can be safely replayed. |
| `upstream[].ip` | Flag to capture and forward the client IP. |
| `upstream[].protocol` | `http` (default), `http2`, `graphql`, `websocket`, `tcp`, or `udp`. HTTP application protocols use the HTTP listener. TCP and UDP use raw listeners. |
| `upstream[].listen` | Local bind address for TCP or UDP upstreams, such as `:7001`. |

### Security

Security settings can be placed under `server.security` for all routes or under an individual `upstream[].security` block. Rate limits are requests per minute; burst values are the initial token capacity. The `RATE_LIMIT` environment variable overrides the global per-IP `rate_limit` value.

```yaml
server:
  security:
    rate_limit: 100
    rate_limit_burst: 20
    global_rate_limit: 1000
    global_rate_burst: 100
    max_body: 10
    allow_cidrs: ["10.0.0.0/8"]
    deny_cidrs: ["10.10.0.0/16"]
    allowed_methods: [GET, POST]
    allowed_paths: ["/api/*"]
    # Set API_KEYS, JWT_SECRET, and REDIS_PASSWORD in the environment.
    require_mtls: false
    redis_addr: "127.0.0.1:6379"
    redis_password: "replace-me"
    redis_db: 0
```

API keys are accepted in `X-API-Key` or as a bearer token. JWT authentication validates HS256 signatures and requires `exp`; `JWT_ISSUER` and `JWT_AUDIENCE` can enforce issuer and audience. mTLS requires the server to be deployed behind TLS with verified client certificates. Request bodies are rejected before proxying when they exceed `max_body` or the upstream `body` limit.

When `redis_addr` is configured, the global limit is also enforced through Redis so multiple LX-2 instances share a minute window. Redis failures return `503` rather than silently bypassing the limit.

Routes are registered only for upstreams with valid, parseable URLs.

## CLI usage

See the [CLI guide](docs/cli.md) for build instructions, command usage, help flags, and runtime file locations.

## PM2

Build the binary and start two Leoxy instances on ports 8080 and 8081:

```bash
go build -o leoxy ./app/cli
pm2 start ecosystem.config.cjs
```

PM2 must use `fork` mode because Leoxy is a Go server. Put Nginx or another load balancer in front of ports 8080 and 8081. The `PORT` environment variable overrides `server.port` for each instance.

On Windows, build `leoxy.exe` and change the ecosystem `script` value to `./leoxy.exe`.

## Endpoints

### Ping

```http
GET /ping
```

Returns `PONG`.

### Liveness

```http
GET /health/live
```

Returns `{"status":"ok"}`.

### Readiness

```http
GET /health/ready
```

Returns `{"status":"ok"}` when at least one server in every configured HTTP upstream group responds without a server error. Returns `503` when any group has no available server. TCP and UDP upstreams are not included.

### Proxy

Each configured upstream is available at its `path` prefix:

```text
/<prefix>/*  ->  upstream server_url
```

The prefix is stripped before forwarding. For example, with the config above, `GET /api/tasks` is forwarded to `http://localhost:7000/tasks`.
