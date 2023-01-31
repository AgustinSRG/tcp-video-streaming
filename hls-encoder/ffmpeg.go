// FFMPEG utilities

package main

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"

	child_process_manager "github.com/AgustinSRG/go-child-process-manager"

	"github.com/vansante/go-ffprobe"
)

var (
	FFMPEG_BINARY_PATH  = "/usr/bin/ffmpeg"  // Location of FFMPEG binary
	FFPROBE_BINARY_PATH = "/usr/bin/ffprobe" // Location of FFPROBE binary
)

// Sets FFMPEG config
// ffmpeg_path - Location of FFMPEG binary
// ffprobe_path - Location of FFPROBE binary
func SetFFMPEGBinaries(ffmpeg_path string, ffprobe_path string) {
	FFMPEG_BINARY_PATH = ffmpeg_path
	FFPROBE_BINARY_PATH = ffprobe_path

	ffprobe.SetFFProbeBinPath(FFPROBE_BINARY_PATH)
}

// Parses frame rate from string returned by ffprobe
// fr - Frame rate in format 'f/t'
func ParseFrameRate(fr string) int32 {
	if fr == "" {
		return 0
	}
	parts := strings.Split(fr, "/")
	if len(parts) == 2 {
		n, err := strconv.Atoi(parts[0])

		if err != nil {
			return 0
		}

		n2, err := strconv.Atoi(parts[1])

		if err != nil || n2 == 0 {
			return 0
		}

		return int32(n) / int32(n2)
	} else if len(parts) == 1 {
		n, err := strconv.Atoi(parts[0])

		if err != nil {
			return 0
		}

		return int32(n)
	} else {
		return 0
	}
}

// Runs FFMPEG command asynchronously (the child process can be managed)
// cmd - Command to run
// input_duration - Duration in seconds (used to calculate progress)
// progress_reporter - Function called each time ffmpeg reports progress via standard error
// Note: If you return true in progress_reporter, the process will be killed (use this to interrupt tasks)
func RunFFMpegCommandAsync(cmd *exec.Cmd, progress_reporter func(time float64, frames uint64) bool) error {
	// Configure command
	err := child_process_manager.ConfigureCommand(cmd)
	if err != nil {
		return err
	}

	// Create a pipe to read StdErr
	pipe, err := cmd.StderrPipe()

	if err != nil {
		return err
	}

	// Start the command

	LogDebug("Running command: " + cmd.String())

	err = cmd.Start()

	if err != nil {
		return err
	}

	// Add process as a child process
	child_process_manager.AddChildProcess(cmd.Process)

	// Read stderr line by line

	reader := bufio.NewReader(pipe)

	var finished bool = false

	for !finished {
		line, err := reader.ReadString('\r')

		if err != nil {
			finished = true
		}

		line = strings.ReplaceAll(line, "\r", "")

		LogDebug("[FFMPEG] " + line)

		if !strings.HasPrefix(line, "frame=") {
			continue // Not a progress line
		}

		// Extract frame

		parts := strings.Split(line, "frame=")

		if len(parts) < 2 {
			continue
		}

		parts = strings.Split(strings.Trim(parts[1], " "), " ")

		if len(parts) < 1 {
			continue
		}

		frames, _ := strconv.ParseUint(parts[0], 10, 64)

		// Extract time

		parts = strings.Split(line, "time=")

		if len(parts) < 2 {
			continue
		}

		parts = strings.Split(strings.Trim(parts[1], " "), " ")

		if len(parts) < 1 {
			continue
		}

		parts = strings.Split(parts[0], ":")

		if len(parts) != 3 {
			continue
		}

		hours, _ := strconv.Atoi(parts[0])
		minutes, _ := strconv.Atoi(parts[1])
		seconds, _ := strconv.ParseFloat(parts[2], 64)

		out_duration := float64(hours)*3600 + float64(minutes)*60 + seconds

		if out_duration > 0 {
			shouldKill := progress_reporter(out_duration, uint64(frames))

			if shouldKill {
				cmd.Process.Kill()
			}
		}
	}

	// Wait for ending

	err = cmd.Wait()

	if err != nil {
		return err
	}

	return nil
}
