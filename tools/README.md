# Leoxy Benchmark Tools

Tools for benchmarking Leoxy (app) - measure throughput, latency, allocations, and save heap profiles.

## Purpose

This tool is only for benchmarking Leoxy. It generates load against a target Leoxy instance and collects:
- Throughput (requests/sec, MB/s)
- Latency percentiles (p50, p95, p99, max)
- Client-side memory allocations and heap usage
- Heap profiles from Leoxy via pprof

## Usage

```bash
go run ./tools/cmd \
  -url http://localhost:4444/api/test \
  -duration 30s \
  -concurrency 100 \
  -pprof http://localhost:4444/debug/pprof/heap \
  -profile-out docs/leoxy-heap.txt
```

## Flags

| Flag | Required | Description |
|-----|----------|-------------|
| `-url` | Yes | Target URL to benchmark |
| `-duration` | Yes | Test duration (e.g., `10s`, `1m`) |
| `-concurrency` | Yes | Number of concurrent workers |
| `-request` | No | Limit total number of requests |
| `-pprof` | No | Leoxy pprof heap URL to fetch (e.g., `http://host:port/debug/pprof/heap`) |
| `-profile-out` | No | Output file for heap profile (default: `docs/leoxy-heap.txt`) |

## Output

The tool prints:
- **Network stats**: target, duration, concurrency, total attempts, success/failed requests, throughput, response throughput, latency percentiles
- **Client memory**: heap in use, total allocated, malloc count (client-side only)
- **Heap profile**: if `-pprof` provided, saves profile and shows analysis command

## Notes

- Client memory stats reflect the benchmark tool process, not Leoxy
- Use `go tool pprof` to analyze saved heap profiles
- For accurate results, run from a separate machine or with adequate resources
- Ensure Leoxy has pprof enabled if fetching profiles

## Building

```bash
go build -o bench ./tools/cmd
```

## Module

```go
module github.com/elrefai99/Leoxy/tools
```
