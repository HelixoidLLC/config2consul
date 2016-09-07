#!/usr/bin/env bash

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o docker/config2consul config2consul.go
chmod +x docker/config2consul
docker build -t config2consul docker

