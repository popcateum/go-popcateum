#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.20.14}"
export CGO_ENABLED=0

if [ "$#" -eq 0 ]; then
    set -- ./cmd/...
fi

for target in windows-amd64 linux-amd64 linux-arm64 darwin-arm64 darwin-amd64; do
    out="build/bin/$target"
    mkdir -p "$out"
    printf 'Building %s in %s\n' "$target" "$out"
    GOOS="${target%-*}" GOARCH="${target#*-}" \
        go build -trimpath -tags urfave_cli_no_docs -o "$out/" "$@"
done
