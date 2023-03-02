// FFMPEG utilities

package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	child_process_manager "github.com/AgustinSRG/go-child-process-manager"

	"gopkg.in/vansante/go-ffprobe.v2"
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

// Prepares a FFMPEG command from the task
// task - Reference to the task
// probeData - Stream metadata (from FFPROBE)
// Returns:
//
//	command - The configured command
//	srcManager - The source manager (optional, can be nil)
//	cmdErr - Error
func PrepareEncodingFFMPEGCommand(task *EncodingTask, probeData *ffprobe.ProbeData) (command *exec.Cmd, srcManager InputSourceManager, cmdErr error) {
	cmd := exec.Command(FFMPEG_BINARY_PATH)

	cmd.Args = make([]string, 1)

	cmd.Args[0] = FFMPEG_BINARY_PATH

	cmd.Args = append(cmd.Args, "-y") // Overwrite

	// Add input source

	sourceManager, err := PrepareEncodingProcessToReceiveSource(cmd, task.sourceType, task.sourceURI)

	if err != nil {
		return nil, nil, err
	}

	videoStream := probeData.FirstVideoStream()
	audioStream := probeData.FirstAudioStream()

	if videoStream == nil {
		return nil, nil, errors.New("The input source does not have a video stream")
	}

	videoWidth := videoStream.Width
	videoHeight := videoStream.Height
	videoFPS := ParseFrameRate(videoStream.AvgFrameRate)

	// Add original output
	if task.resolutions.hasOriginal {
		if videoStream.CodecName == "h264" && (audioStream == nil || audioStream.CodecName == "aac") {
			// No need to re-encode, copy the stream

			cmd.Args = append(cmd.Args, "-vcodec", "copy", "-acodec", "copy")
		} else {
			// Encode

			cmd.Args = append(cmd.Args, "-vcodec", "libx264", "-acodec", "aac")
			cmd.Args = append(cmd.Args, "-vf", "pad=ceil(iw/2)*2:ceil(ih/2)*2") // Ensure even width and height
		}
		AppendGenericHLSArguments(cmd, Resolution{width: videoWidth, height: videoHeight, fps: videoFPS}, task)
	}

	// Add resized outputs
	resolutions := GetActualResolutionList(Resolution{width: videoWidth, height: videoHeight, fps: videoFPS}, task.resolutions)
	for i := 0; i < len(resolutions); i++ {
		cmd.Args = append(cmd.Args, "-vcodec", "libx264", "-acodec", "aac")

		videoFilter := ""

		if resolutions[i].fps >= 0 && resolutions[i].fps != videoFPS {
			videoFilter += "fps=" + fmt.Sprint(resolutions[i].fps) + ","
		}

		videoFilter += "scale=" + fmt.Sprint(resolutions[i].width) + ":" + fmt.Sprint(resolutions[i].height)

		cmd.Args = append(cmd.Args, "-vf", videoFilter) // Scale and FPS

		AppendGenericHLSArguments(cmd, resolutions[i], task)
	}

	// Add previews (if enabled)

	if task.previews.enabled {
		cmd.Args = append(cmd.Args, "-f", "image2")

		videoFilter := "fps=1/" + fmt.Sprint(task.previews.delaySeconds) +
			",scale=" + fmt.Sprint(task.previews.width) + ":" + fmt.Sprint(task.previews.height) +
			":force_original_aspect_ratio=decrease,pad=" + fmt.Sprint(task.previews.width) + ":" + fmt.Sprint(task.previews.height) +
			":(ow-iw)/2:(oh-ih)/2"

		cmd.Args = append(cmd.Args, "-vf", videoFilter)

		cmd.Args = append(cmd.Args, "-protocol_opts", "method=PUT")

		cmd.Args = append(cmd.Args, "-start_number", "0")

		cmd.Args = append(cmd.Args, "http://127.0.0.1:"+fmt.Sprint(task.server.loopBackPort)+"/img-preview/"+task.channel+"/"+task.streamId+"/"+task.previews.Encode("-")+"/%d.jpg")
	}

	return cmd, sourceManager, nil
}

// Parses frame rate from string returned by ffprobe
// fr - Frame rate in format 'f/t'
func ParseFrameRate(fr string) int {
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

		return n / n2
	} else if len(parts) == 1 {
		n, err := strconv.Atoi(parts[0])

		if err != nil {
			return 0
		}

		return n
	} else {
		return 0
	}
}

// Reads progress and calls a reporter
// pipe - Stderr pipe
// progress_reporter - Function called each time ffmpeg reports progress via standard error
func ReadFFMPEGProgress(pipe io.ReadCloser, progress_reporter func(time float64, frames uint64)) {
	reader := bufio.NewReader(pipe)

	var finished bool = false

	for !finished {
		line, err := reader.ReadString('\r')

		if err != nil {
			finished = true
		}

		line = strings.ReplaceAll(line, "\r", "")

		LogTrace("[FFMPEG] " + line)

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
			progress_reporter(out_duration, uint64(frames))
		}
	}
}

// Runs FFMPEG command asynchronously (the child process can be managed)
// cmd - Command to run
// input_duration - Duration in seconds (used to calculate progress)
// progress_reporter - Function called each time ffmpeg reports progress via standard error
func RunFFMpegCommandAsync(cmd *exec.Cmd, progress_reporter func(time float64, frames uint64)) (process *os.Process, cmdErr error) {
	// Configure command
	err := child_process_manager.ConfigureCommand(cmd)
	if err != nil {
		return nil, err
	}

	// Create a pipe to read StdErr
	pipe, err := cmd.StderrPipe()

	if err != nil {
		return nil, err
	}

	// Start the command

	err = cmd.Start()

	if err != nil {
		return nil, err
	}

	// Add process as a child process
	child_process_manager.AddChildProcess(cmd.Process)

	// Read stderr line by line

	go ReadFFMPEGProgress(pipe, progress_reporter)

	// Return process

	return cmd.Process, nil
}
