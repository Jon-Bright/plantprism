#!/bin/bash
VERSION=$(git rev-parse HEAD |cut -c 33-)
echo "Using version '${VERSION}'"
go build -ldflags "-X main.GitVersion=${VERSION}"
