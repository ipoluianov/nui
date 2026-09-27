ECHO Building for macOS ARM64...
SET GOOS=darwin
SET GOARCH=arm64
go build -o ../../../bin/macos_arm64/nui ../../../main.go
