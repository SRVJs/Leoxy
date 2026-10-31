# =============== #
# Stage 1: Build  #
# =============== #
FROM golang:1.27.3 AS builder

WORKDIR /src

COPY app/go.mod app/go.sum ./app/
RUN cd app && go mod download

COPY app ./app
RUN cd app && CGO_ENABLED=0 GOOS=linux GOWORK=off go build -trimpath -ldflags="-s -w" -o /out/leoxy ./cli


# =================== #
# Stage 2: Production #
# =================== #
FROM alpine:3.24

RUN addgroup -S leoxy \
    && adduser -S -G leoxy -h /app leoxy \
    && mkdir -p /app/leoxy \
    && chown -R leoxy:leoxy /app

WORKDIR /app

COPY --from=builder /out/leoxy /usr/local/bin/leoxy
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENV LEOXY_HOST=0.0.0.0 \
    LEOXY_LOG_DIR=/app/leoxy/log

USER leoxy

EXPOSE 4444

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:${PORT:-4444}/health/live || exit 1

ENTRYPOINT ["/usr/local/bin/leoxy"]
