#!/usr/bin/env bash

# Echo out all commands for monitoring progress
set -x

# When using the provided.al2 runtime, the binary must be named "bootstrap" and be in the root directory
CGO_ENABLED=0 go build -C src -tags lambda.norpc -ldflags="-s -w" -o bin/bootstrap && \
cp src/config.json src/bin/config.json
