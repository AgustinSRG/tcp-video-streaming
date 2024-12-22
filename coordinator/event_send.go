// Event sending system

package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	EVENT_SEND_RETRY_DELAY = 10 * time.Second
)

// Sends an stream-available event
// Retries until success
// channel - Reference to the channel
// event - Reference to the event
func SendStreamAvailableEvent(channel *StreamingChannel, event *PendingStreamAvailableEvent) {
	sent := false

	eventURL := os.Getenv("EVENT_CALLBACK_URL")

	if eventURL == "" {
		LogWarning("No EVENT_CALLBACK_URL set. Ignoring stream-available event.")
		sent = true
	}

	authorization := ""

	authMethod := strings.ToUpper(os.Getenv("EVENT_CALLBACK_AUTH"))

	switch authMethod {
	case "BASIC":
		user := os.Getenv("EVENT_CALLBACK_AUTH_USER")
		password := os.Getenv("EVENT_CALLBACK_PASSWORD")
		authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
	case "BEARER":
		token := os.Getenv("EVENT_CALLBACK_AUTH_TOKEN")
		authorization = "Bearer " + token
	case "CUSTOM":
		authorization = os.Getenv("EVENT_CALLBACK_AUTH_CUSTOM")
	}

	for !sent && !event.cancelled {
		client := &http.Client{}

		req, e := http.NewRequest("POST", eventURL, nil)

		if e != nil {
			LogError(e)
			time.Sleep(EVENT_SEND_RETRY_DELAY)
			continue
		}

		req.Header.Set("x-streaming-channel", event.channel)
		req.Header.Set("x-streaming-id", event.streamId)
		req.Header.Set("x-event-type", "stream-available")
		req.Header.Set("x-stream-type", event.streamType)
		req.Header.Set("x-resolution", event.resolution)
		req.Header.Set("x-index-file", event.indexFile)

		if event.startTime != "" {
			req.Header.Set("x-start-time", event.startTime)
		}

		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}

		LogDebug("Sending stream-available for: " + event.channel + ":" + event.streamId + " / POST: " + eventURL)

		res, e := client.Do(req)

		if e != nil {
			LogError(e)
			time.Sleep(EVENT_SEND_RETRY_DELAY)
			continue
		}

		if res.StatusCode == 200 {
			sent = true
		} else {
			LogDebug("[" + event.channel + ":" + event.streamId + "] [Stream-available] [Error] Could not send event. Status code: " + fmt.Sprint(res.StatusCode))
			time.Sleep(EVENT_SEND_RETRY_DELAY)
		}
	}

	// Remove event from the list

	channel.mutex.Lock()
	delete(channel.pendingEvents, event.id)
	channel.mutex.Unlock()
}

// Sends an stream-closed event
// Retries until success
// coordinator - Reference to the coordinator
// event - Reference to the event
func SendStreamClosedEvent(coordinator *Streaming_Coordinator, event *PendingStreamClosedEvent) {
	sent := false

	eventURL := os.Getenv("EVENT_CALLBACK_URL")

	if eventURL == "" {
		LogWarning("No EVENT_CALLBACK_URL set. Ignoring stream-available event.")
		sent = true
	}

	authorization := ""

	authMethod := strings.ToUpper(os.Getenv("EVENT_CALLBACK_AUTH"))

	switch authMethod {
	case "BASIC":
		user := os.Getenv("EVENT_CALLBACK_AUTH_USER")
		password := os.Getenv("EVENT_CALLBACK_PASSWORD")
		authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
	case "BEARER":
		token := os.Getenv("EVENT_CALLBACK_AUTH_TOKEN")
		authorization = "Bearer " + token
	case "CUSTOM":
		authorization = os.Getenv("EVENT_CALLBACK_AUTH_CUSTOM")
	}

	for !sent && !event.cancelled {
		client := &http.Client{}

		req, e := http.NewRequest("POST", eventURL, nil)

		if e != nil {
			LogError(e)
			time.Sleep(EVENT_SEND_RETRY_DELAY)
			continue
		}

		req.Header.Set("x-streaming-channel", event.channel)
		req.Header.Set("x-streaming-id", event.streamId)
		req.Header.Set("x-event-type", "stream-closed")

		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}

		LogDebug("Sending stream-closed for: " + event.channel + ":" + event.streamId + " / POST: " + eventURL)

		res, e := client.Do(req)

		if e != nil {
			LogError(e)
			time.Sleep(EVENT_SEND_RETRY_DELAY)
			continue
		}

		if res.StatusCode == 200 {
			sent = true
		} else {
			LogDebug("[" + event.channel + ":" + event.streamId + "] [Stream-closed] [Error] Could not send event. Status code: " + fmt.Sprint(res.StatusCode))
			time.Sleep(EVENT_SEND_RETRY_DELAY)
		}
	}

	// Call coordinator method to indicate the event being sent
	coordinator.RemoveActiveStream(event.channel, event.streamId)
}
