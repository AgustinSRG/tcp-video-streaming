@echo off

call go build -ldflags="-s -w" -o hls-encoder.exe
