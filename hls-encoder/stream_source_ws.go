// Stream source utils (For WebSocket sources)

package main

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/gorilla/websocket"
	"gopkg.in/vansante/go-ffprobe.v2"
)

// Probes WebSocket source
// sourceURI - WebSocket URI
// Returns
//  probeData - Video stream probe data
//  err - Error
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
