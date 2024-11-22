// Watch API

package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type ChannelStatusResponse struct {
	Id string `json:"id"`

	Record      bool   `json:"record"`
	Resolutions string `json:"resolutions"`
	Previews    string `json:"previews"`

	Live     bool   `json:"live"`
	StreamId string `json:"streamId"`

	LiveStartTimestamp int64 `json:"liveStartTimestamp"`

	LiveSubStreams []SubStreamWithCdn `json:"liveSubStreams"`
}

func api_getChannelStatus(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)

	channel := vars["channel"]

	if !validateStreamIDString(channel) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	result := DATABASE.GetChannelStatus(channel)

	if result == nil {
		ReturnAPIError(response, 404, "NOT_FOUND", "Channel not found.")
		return
	}

	jsonResult, err := json.Marshal(result)

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
		return
	}

	ReturnAPI_JSON(response, request, jsonResult)
}

type VODItem struct {
	StreamId  string `json:"streamId"`
	Timestamp int64  `json:"timestamp"`
}

type ListVODResponse struct {
	VODS []VODItem `json:"vod_list"`
}

func api_listVODs(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)

	channel := vars["channel"]

	if !validateStreamIDString(channel) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	result := ListVODResponse{
		VODS: DATABASE.GetChannelVODList(channel),
	}

	if result.VODS == nil {
		ReturnAPIError(response, 404, "NOT_FOUND", "Channel not found.")
		return
	}

	jsonResult, err := json.Marshal(result)

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
		return
	}

	ReturnAPI_JSON(response, request, jsonResult)
}

func api_getVOD(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)

	channel := vars["channel"]
	streamId := vars["vod"]

	if !validateStreamIDString(channel) || !validateStreamIDString(streamId) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	result := DATABASE.GetChannelVOD(channel, streamId)

	if result == nil {
		ReturnAPIError(response, 404, "NOT_FOUND", "Channel not found.")
		return
	}

	jsonResult, err := json.Marshal(result)

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
		return
	}

	ReturnAPI_JSON(response, request, jsonResult)
}
