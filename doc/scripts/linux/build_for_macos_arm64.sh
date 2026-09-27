#!/bin/bash
export GOOS=darwin
export GOARCH=arm64
export CGO_ENABLED=0
go build -o ../../../bin/macos_arm64/nui ../../../main.go
