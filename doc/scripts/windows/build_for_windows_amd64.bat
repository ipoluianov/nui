echo Building for Windows AMD64...
SET GOOS=windows
SET GOARCH=amd64
go build -ldflags -H=windowsgui -o ../../../bin/windows_amd64/nui.exe ../../../main.go
