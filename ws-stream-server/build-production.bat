@echo off

call go build -ldflags="-s -w" -o ws-stream-server.exe
