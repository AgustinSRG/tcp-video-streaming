// Stream source utils (For WebSocket sources)

package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gopkg.in/vansante/go-ffprobe.v2"
)

// Probes WebSocket source
// sourceURI - WebSocket URI
// Returns
//
//	probeData - Video stream probe data
//	err - Error
func ProbeStreamSource_WS(sourceURI string) (probeData *ffprobe.ProbeData, err error) {
	// Get first video chunk from websocket connection
	conn, _, err := websocket.DefaultDialer.Dial(sourceURI+"/probe", nil)

	if err != nil {
		return nil, err
	}

	deadLine := time.Now().Add(SOURCE_PROBE_TIMEOUT)
	probed := false
	firstVideoChunk := make([]byte, 0)

	for !probed {
		err = conn.SetReadDeadline(deadLine)

		if err != nil {
			conn.Close()
			return nil, err
		}

		msgType, message, err := conn.ReadMessage()

		if err != nil {
			conn.Close()
			return nil, err
		}

		if msgType == websocket.BinaryMessage {
			probed = true
			firstVideoChunk = message
		} else {
			if time.Now().UnixMilli() > deadLine.UnixMilli() {
				conn.Close()
				return nil, errors.New("WebSocket probe connection timed out")
			} else {
				conn.WriteMessage(websocket.TextMessage, []byte("h"))
			}
		}
	}

	conn.Close()

	// Probe first video chunk

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	data, err := ffprobe.ProbeReader(ctx, bytes.NewReader(firstVideoChunk))
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Manager to receive WebSocket video stream
type WebSocketInputManager struct {
	sourceURI string
	conn      *websocket.Conn
	stdin     io.WriteCloser
	mutex     *sync.Mutex
}

// Prepares an encoding process to receive a video stream source from WebSocket
// cmd - A reference to the command (will be configured)
// sourceURI - WebSocket URI
// Returns
//
//	manager - A manager to control the source
//	err - Error
func PrepareEncodingProcessToReceiveSource_WS(cmd *exec.Cmd, sourceURI string) (manager InputSourceManager, err error) {
	cmd.Args = append(cmd.Args, "-i", "-")

	stdin, err := cmd.StdinPipe()

	if err != nil {
		return nil, err
	}

	mng := &WebSocketInputManager{
		sourceURI: sourceURI,
		conn:      nil,
		stdin:     stdin,
		mutex:     &sync.Mutex{},
	}

	return mng, nil
}

// Starts receiving the Websocket source
func (mng *WebSocketInputManager) Start() error {
	conn, _, err := websocket.DefaultDialer.Dial(mng.sourceURI+"/receive-clear-cache", nil)

	if err != nil {
		mng.stdin.Close()
		return err
	}

	mng.conn = conn

	go mng.RunReaderLoop(conn)

	return nil
}

// Reads chunks from WebSocket and writes it to the stdin of the encoding process
// conn - Connection
func (mng *WebSocketInputManager) RunReaderLoop(conn *websocket.Conn) {
	err := conn.WriteMessage(websocket.TextMessage, []byte("h"))

	if err != nil {
		mng.Close()
		return
	}

	lastHeartBeat := time.Now().UnixMilli()

	for {
		err = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		if err != nil {
			mng.Close()
			return
		}

		msgType, message, err := conn.ReadMessage()

		if err != nil {
			mng.Close()
			return
		}

		if msgType == websocket.BinaryMessage {
			// Received a chunk, write it
			err = mng.WriteChunk(message)

			if err != nil {
				mng.Close()
				return
			}

			if time.Now().UnixMilli()-lastHeartBeat > (20 * 1000) {
				// Send heartbeat
				err = conn.WriteMessage(websocket.TextMessage, []byte("h"))
				if err != nil {
					mng.Close()
					return
				}
				lastHeartBeat = time.Now().UnixMilli()
			}
		} else {
			// Send heartbeat
			err = conn.WriteMessage(websocket.TextMessage, []byte("h"))
			if err != nil {
				mng.Close()
				return
			}
			lastHeartBeat = time.Now().UnixMilli()
		}
	}
}

// Writes a chunk to the encoding process
// chunk - The chunk
func (mng *WebSocketInputManager) WriteChunk(chunk []byte) error {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()

	_, err := mng.stdin.Write(chunk)

	return err
}

// Closes the WebSocket source
func (mng *WebSocketInputManager) Close() {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()

	if mng.conn != nil {
		mng.conn.Close()
	}

	mng.stdin.Close()
}
