FROM golang:1.27.3-alpine AS builder

WORKDIR /src

COPY tools/go.mod tools/go.sum ./tools/
RUN cd tools && go mod download

COPY tools ./tools
RUN cd tools && CGO_ENABLED=0 GOOS=linux GOWORK=off go build -trimpath -ldflags="-s -w" -o /out/leoxy-bench ./cmd

FROM alpine:3.22

RUN adduser -S -H bench

COPY --from=builder /out/leoxy-bench /usr/local/bin/leoxy-bench

USER bench

ENTRYPOINT ["/usr/local/bin/leoxy-bench"]
