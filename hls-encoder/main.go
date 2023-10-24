// Main

package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/joho/godotenv"

	child_process_manager "github.com/AgustinSRG/go-child-process-manager"
)

const VERSION = "1.0.0"

func main() {
	godotenv.Load() // Load env vars

	InitLog()

	InitFileStorage()

	LogInfo("Started HLS encoder worker - Version " + VERSION)

	err := child_process_manager.InitializeChildProcessManager()
	if err != nil {
		fmt.Println("Error: " + err.Error())
		os.Exit(1)
	}
	defer child_process_manager.DisposeChildProcessManager()

	// Load config

	ffmpegPath := os.Getenv("FFMPEG_PATH")

	if ffmpegPath == "" {
		if runtime.GOOS == "windows" {
			ffmpegPath = "/ffmpeg/bin/ffmpeg.exe"
		} else {
			ffmpegPath = "/usr/bin/ffmpeg"
		}
	}

	if _, err := os.Stat(ffmpegPath); err != nil {
		fmt.Println("Error: Could not find 'ffmpeg' at specified location: " + ffmpegPath)
		os.Exit(1)
		return
	}

	ffprobePath := os.Getenv("FFPROBE_PATH")

	if ffprobePath == "" {
		if runtime.GOOS == "windows" {
			ffprobePath = "/ffmpeg/bin/ffprobe.exe"
		} else {
			ffprobePath = "/usr/bin/ffprobe"
		}
	}

	if _, err := os.Stat(ffprobePath); err != nil {
		fmt.Println("Error: Could not find 'ffprobe' at specified location: " + ffprobePath)
		os.Exit(1)
		return
	}

	SetFFMPEGBinaries(ffmpegPath, ffprobePath) // Set FFMPEG paths

	// Start server

	server := &HLS_Encoder_Server{}

	server.Initialize()
	server.Start()
}
