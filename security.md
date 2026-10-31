# Leoxy security notes

## Rate limits

`rate_limit`, `global_rate_limit` and `route_rate_limit` are requests per minute. The burst values are the bucket size. IPv6 clients are limited per /64.

## Client IP

Set `trusted_proxy_cidrs` to the addresses of your TLS terminator. Without it every client appears as the proxy address and per-IP limits and CIDR rules apply to the proxy only. Leoxy reads `X-Forwarded-For` first and uses `X-Real-IP` only when that header is absent.

## Internal endpoints

`/metrics`, `/health/ready` and `/leoxy` answer only to loopback and to `internal_cidrs`, and only when the request carries no forwarding headers. Also block them at the terminator:

```nginx
location ~ ^/(metrics|health/ready|leoxy)$ { return 404; }
```

`/ping` and `/health/live` are public and bypass authentication and rate limits.

## Authentication

`jwt_secret` must be at least 32 bytes. Set `jwt_issuer` and `jwt_audience` when tokens come from a shared identity provider. `api_keys` must not contain empty values. Keep secrets in environment variables.

## mTLS

`require_mtls` needs `tls_cert`, `tls_key` and `client_ca` on `server.security`. Leoxy then terminates TLS itself. Behind a TLS terminator, enforce client certificates there.

## TCP and UDP listeners

They bypass HTTP authentication. Set `allow_cidrs` and `max_conns_per_ip` on every stream upstream. UDP source addresses can be spoofed, so never expose a UDP listener to the internet.

## Redis limiter

`redis_fail_open: true` lets traffic through when Redis is down. The default rejects with 503.

## Container

Bind the published port to loopback or a private interface and put the TLS terminator in front. The data directory must be writable by the container user.
