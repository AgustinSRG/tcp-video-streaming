// Task main routine

package main

// Runs the task
func (task *EncodingTask) Run() {
}

// Kills the task
func (task *EncodingTask) Kill() {
	task.mutex.Lock()
	defer task.mutex.Unlock()

	task.killed = true

	if task.process != nil {
		task.process.Kill()
	}
}
