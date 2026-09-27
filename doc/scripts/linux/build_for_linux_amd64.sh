#!/bin/bash
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0
go build -o ../../../bin/linux_amd64/nui ../../../main.go
