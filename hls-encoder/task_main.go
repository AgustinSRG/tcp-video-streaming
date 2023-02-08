// Task main routine

package main

import (
	"fmt"
	"strings"
)

// Runs the task
func (task *EncodingTask) Run() {
	defer func() {
		if err := recover(); err != nil {
			switch x := err.(type) {
			case string:
				task.log("Error: " + x)
			case error:
				task.log("Error: " + x.Error())
			default:
				task.log("Task Crashed!")
			}
		}
		task.log("Task ended.")
		// Announce closed
		task.server.websocketControlConnection.SendStreamClosed(task.channel, task.streamId)
		// Remove task
		task.server.RemoveTask(task.channel, task.streamId)
	}()

	// Probe stream

	task.debug("Probing stream source: " + task.sourceType + " - " + task.sourceURI)

	probeData, err := ProbeStreamSource(task.sourceType, task.sourceURI)

	if err != nil {
		task.log("Error: " + err.Error())
		return
	}

	if task.killed {
		return
	}

	// Create encoding process

	cmd, srcManager, err := PrepareEncodingFFMPEGCommand(task, probeData)

	if err != nil {
		task.log("Error: " + err.Error())
		return
	}

	task.debug("Command to be run: " + strings.Join(cmd.Args, " "))

	process, err := RunFFMpegCommandAsync(cmd, func(time float64, frames uint64) {
		task.OnEncodingProgress()
		task.debug("[FFMPEG-P] TIME=" + fmt.Sprint(time) + ", FRAMES=" + fmt.Sprint(frames))
	})

	if err != nil {
		task.log("Error: " + err.Error())
		return
	}

	err = srcManager.Start()

	if err != nil {
		process.Kill()
		task.log("Error: " + err.Error())
		return
	}

	task.mutex.Lock()

	if task.killed {
		srcManager.Close()
		task.mutex.Unlock()
		return
	}

	task.process = process

	task.mutex.Unlock()

	state, err := process.Wait()

	srcManager.Close()

	if err != nil {
		task.log("Error: " + err.Error())
		return
	}

	task.debug("FFMPEG ended with state: " + state.String())

	task.OnEncodingEnded()
}

// Kills the task
func (task *EncodingTask) Kill() {
	task.mutex.Lock()
	defer task.mutex.Unlock()

	task.killed = true

	if task.process != nil && !task.hasStarted {
		task.process.Kill()
	}
}
