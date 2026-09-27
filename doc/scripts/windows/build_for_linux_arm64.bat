echo Building for Linux ARM64...
SET GOOS=linux
SET GOARCH=arm64
go build -o ../../../bin/linux_arm64/nui ../../../main.go
