#!/usr/bin/env sh
set -eu

goos="${1:-linux}"
goarch="${2:-amd64}"
goarm="${3:-}"
extension=""

[ "$goos" = windows ] && extension=".exe"

arch="$goarch"

[ -n "$goarm" ] && arch="${goarch}v${goarm}"

mkdir -p dist

CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" GOARM="$goarm" \
  go build -trimpath -ldflags="-s -w" -o "dist/leoxy-$goos-$arch$extension" ./app/cli
