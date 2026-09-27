#!/bin/bash
export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=0
go build -ldflags -H=windowsgui -o ../../../bin/windows_amd64/nui.exe ../../../main.go
