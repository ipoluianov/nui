echo Building for Linux AMD64...
SET GOOS=linux
SET GOARCH=amd64
go build -o ../../../bin/linux_amd64/nui ../../../main.go
