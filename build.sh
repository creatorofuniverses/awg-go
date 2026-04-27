#!/usr/bin/env sh
set -eu
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}
exec go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o awg-go .
