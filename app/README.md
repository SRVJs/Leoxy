# LX-2 App

Core application code for the LX-2 reverse proxy.

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
