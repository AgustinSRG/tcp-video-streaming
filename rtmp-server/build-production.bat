@echo off

call go build -ldflags="-s -w" -o rtmp-server.exe
