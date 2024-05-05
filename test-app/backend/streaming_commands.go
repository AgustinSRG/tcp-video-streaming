// Send streaming commands to the coordinator

package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func sendStreamingCommand(method string, path string, headers map[string]string) (resErr error, resBody []byte) {
	baseURL := os.Getenv("CONTROL_SERVER_BASE_URL")

	if baseURL == "" {
		return errors.New("CONTROL_SERVER_BASE_URL not set"), nil
	}

	realURL, err := url.JoinPath(baseURL, path)

	if err != nil {
		return err, nil
	}

	authorization := ""

	authMethod := strings.ToUpper(os.Getenv("STREAMING_COMMANDS_AUTH"))

	switch authMethod {
	case "BASIC":
		user := os.Getenv("STREAMING_COMMANDS_AUTH_USER")
		password := os.Getenv("STREAMING_COMMANDS_PASSWORD")
		authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
	case "BEARER":
		token := os.Getenv("STREAMING_COMMANDS_AUTH_TOKEN")
		authorization = "Bearer " + token
	case "CUSTOM":
		authorization = os.Getenv("STREAMING_COMMANDS_AUTH_CUSTOM")
	}

	client := &http.Client{}

	req, e := http.NewRequest(method, realURL, nil)

	if e != nil {
		return e, nil
	}

	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	if headers != nil {
		for key, val := range headers {
			req.Header.Set(key, val)
		}
	}

	LogDebug("Sending streaming command: " + realURL)

	res, e := client.Do(req)

	if e != nil {
		LogError(e)
		return e, nil
	}

	defer res.Body.Close()

	if res.StatusCode == 200 {
		rBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err, nil
		}
		return nil, rBody
	} else {
		return errors.New("Status code: " + fmt.Sprint(res.StatusCode)), nil
	}
}

// Response for the capacity API
type CapacityAPIResponse struct {
	Load         int `json:"load"`
	Capacity     int `json:"capacity"`
	EncoderCount int `json:"encoderCount"`
}

func StreamingCommands_GetCapacity() (error, *CapacityAPIResponse) {
	res := CapacityAPIResponse{}

	err, resBody := sendStreamingCommand("GET", "/commands/capacity", nil)

	if err != nil {
		return err, nil
	}

	err = json.Unmarshal(resBody, &res)

	if err != nil {
		return err, nil
	}

	return nil, &res
}

type ReportAPIResponse_StreamingServer struct {
	Id         uint64 `json:"id"`
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	SSL        bool   `json:"ssl"`
	ServerType string `json:"serverType"`
}

type ReportAPIResponse_Encoder struct {
	Id       uint64 `json:"id"`
	Capacity int    `json:"capacity"`
	Load     int    `json:"load"`
}

type ReportAPIResponse_ActiveStream struct {
	Channel      string `json:"channel"`
	StreamId     string `json:"streamId"`
	StreamServer uint64 `json:"streamServer"`
	Encoder      uint64 `json:"encoder"`
}

type ReportAPIResponse struct {
	StreamingServers []ReportAPIResponse_StreamingServer `json:"streamingServers"`
	Encoders         []ReportAPIResponse_Encoder         `json:"encoders"`
	ActiveStreams    []ReportAPIResponse_ActiveStream    `json:"activeStreams"`
}

func StreamingCommands_GetReport() (error, *ReportAPIResponse) {
	res := ReportAPIResponse{}

	err, resBody := sendStreamingCommand("GET", "/commands/report", nil)

	if err != nil {
		return err, nil
	}

	err = json.Unmarshal(resBody, &res)

	if err != nil {
		return err, nil
	}

	return nil, &res
}

func StreamingCommands_CloseStream(channel string, streamId string) error {
	err, _ := sendStreamingCommand("POST", "/commands/close", map[string]string{
		"x-streaming-channel": channel,
		"x-streaming-id":      streamId,
	})

	if err != nil {
		return err
	}

	return nil
}
