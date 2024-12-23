// Events handler callback

package main

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func checkEventsCallbackAuth(auth string) bool {
	authMethod := strings.ToUpper(os.Getenv("EVENT_CALLBACK_AUTH"))
	authorization := ""

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

	return subtle.ConstantTimeCompare([]byte(authorization), []byte(auth)) == 1
}

func callback_eventsHandler(response http.ResponseWriter, request *http.Request) {
	if !checkEventsCallbackAuth(request.Header.Get("Authorization")) {
		ReturnAPIError(response, 401, "UNAUTHORIZED", "Invalid authorization.")
		return
	}

	channel := request.Header.Get("x-streaming-channel")
	streamId := request.Header.Get("x-streaming-id")
	eventType := request.Header.Get("x-event-type")
	streamType := request.Header.Get("x-stream-type")
	resolution := request.Header.Get("x-resolution")
	indexFile := request.Header.Get("x-index-file")

	startTime := float64(0)

	if request.Header.Get("x-start-time") != "" {
		st, err := strconv.ParseFloat(request.Header.Get("x-start-time"), 64)

		if err == nil {
			startTime = st
		}
	}

	LogDebug("Incoming streaming event. Channel=" + channel + ", ID=" + streamId + ", EV_TYPE=" + eventType + ", S_TYPE=" + streamType + ", RESOLUTION=" + resolution + ", INDEX=" + indexFile + ", START_TIME=" + fmt.Sprint(startTime))

	var err error = nil

	if eventType == "stream-available" {
		err = DATABASE.AddAvailableStream(channel, streamId, streamType, resolution, indexFile, startTime)
	} else if eventType == "stream-closed" {
		err = DATABASE.CloseStream(channel, streamId)
	}

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
		return
	}

	err = DATABASE.Save()

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
		return
	}

	response.WriteHeader(200)
	response.Write([]byte("OK"))
}
