# LX-2 App

Core application code for the LX-2 reverse proxy.

## Structure

```
app/
├── cli/           # CLI entrypoint (cobra commands)
├── leoxy/         # Runtime directory (config, logs)
└── pkg/
    ├── internal/  # Internal packages
    │   ├── common/      # Logging, metrics, IP utilities
    │   ├── file/        # Config loading
    │   ├── http_server/ # HTTP, TCP, UDP servers and routing
    │   └── security/    # Auth, rate limiting, body limits
    └── server/     # Server orchestration
```

## Building

```bash
go build -o leoxy ./app/cli
```

## Running

```bash
./leoxy             # Foreground
./leoxy start       # Background
./leoxy stop        # Stop
./leoxy version     # Version
```

## Module

```go
module github.com/elrefai99/Leoxy/app
```

## Version

Current version: `v3.8.2` (defined in `app/cli/main.go`)
