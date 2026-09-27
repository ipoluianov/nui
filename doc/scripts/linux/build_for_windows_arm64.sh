#!/bin/bash
export GOOS=windows
export GOARCH=arm64
export CGO_ENABLED=0
go build -ldflags -H=windowsgui -o ../../../bin/windows_arm64/nui.exe ../../../main.go
