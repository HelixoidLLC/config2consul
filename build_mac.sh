#!/usr/bin/env bash
set -e
CGO_ENABLED=0 GOOS=darwin go build -a -installsuffix cgo -o config2consul_mac config2consul.go
chmod +x config2consul_mac