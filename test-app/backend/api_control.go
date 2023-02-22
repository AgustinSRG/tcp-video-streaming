// Control API

package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

type PublishingDetailsAPIResponse struct {
	RTMP_BASE_URL string `json:"rtmp_base_url"`
	WSS_BASE_URL  string `json:"wss_base_url"`
}

func api_getPublishingDetails(response http.ResponseWriter, request *http.Request) {
	rtmpBaseURL := os.Getenv("RTMP_BASE_URL")
	wssBaseURL := os.Getenv("WSS_BASE_URL")

	result := PublishingDetailsAPIResponse{
		RTMP_BASE_URL: rtmpBaseURL,
		WSS_BASE_URL:  wssBaseURL,
	}

	jsonResult, err := json.Marshal(result)

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
		return
	}

	ReturnAPI_JSON(response, request, jsonResult)
}

type ChannelCreateBody struct {
	Id string `json:"id"`

	Record      bool   `json:"record"`
	Resolutions string `json:"resolutions"`
	Previews    string `json:"previews"`
}

type ChannelChangedResponse struct {
	Id  string `json:"id"`
	Key string `json:"key"`

	Record      bool   `json:"record"`
	Resolutions string `json:"resolutions"`
	Previews    string `json:"previews"`
}

func api_createChannel(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, JSON_BODY_MAX_LENGTH)

	var p ChannelCreateBody

	err := json.NewDecoder(request.Body).Decode(&p)
	if err != nil {
		response.WriteHeader(400)
		return
	}

	if !validateStreamIDString(p.Id) {
		ReturnAPIError(response, 400, "INVALID_CHANNEL_ID", "")
		return
	}

	resolutions := DecodeResolutionsList(p.Resolutions)

	for i := 0; i < len(resolutions.resolutions); i++ {
		if resolutions.resolutions[i].width < 16 || resolutions.resolutions[i].height < 16 {
			ReturnAPIError(response, 400, "INVALID_RESOLUTIONS", "")
			return
		}
	}

	previews := DecodePreviewsConfiguration(p.Previews, ",")

	if previews.enabled && (previews.width < 16 || previews.height < 16 || previews.delaySeconds < 1) {
		ReturnAPIError(response, 400, "INVALID_PREVIEWS_CONFIG", "")
		return
	}

	result, success := DATABASE.CreateChannel(p.Id, p.Record, resolutions, previews)

	if !success {
		ReturnAPIError(response, 400, "ID_TAKEN", "")
		return
	}

	err = DATABASE.Save()

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
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

type ChannelConfigBody struct {
	Key string `json:"key"`

	Record      bool   `json:"record"`
	Resolutions string `json:"resolutions"`
	Previews    string `json:"previews"`
}

func api_configChannel(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	request.Body = http.MaxBytesReader(response, request.Body, JSON_BODY_MAX_LENGTH)

	var p ChannelConfigBody

	err := json.NewDecoder(request.Body).Decode(&p)
	if err != nil {
		response.WriteHeader(400)
		return
	}

	channel := vars["channel"]

	if !validateStreamIDString(channel) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	valid, _, _, _ := DATABASE.VerifyKey(channel, p.Key)

	if !valid {
		ReturnAPIError(response, 403, "INVALID_KEY", "Access denied")
		return
	}

	resolutions := DecodeResolutionsList(p.Resolutions)

	for i := 0; i < len(resolutions.resolutions); i++ {
		if resolutions.resolutions[i].width < 16 || resolutions.resolutions[i].height < 16 {
			ReturnAPIError(response, 400, "INVALID_RESOLUTIONS", "")
			return
		}
	}

	previews := DecodePreviewsConfiguration(p.Previews, ",")

	if previews.enabled && (previews.width < 16 || previews.height < 16 || previews.delaySeconds < 1) {
		ReturnAPIError(response, 400, "INVALID_PREVIEWS_CONFIG", "")
		return
	}

	result, success := DATABASE.UpdateChannel(channel, p.Record, resolutions, previews)

	if !success {
		ReturnAPIError(response, 404, "CHANNEL_NOT_FOUND", "")
		return
	}

	err = DATABASE.Save()

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
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

type ChannelActionBody struct {
	Key string `json:"key"`
}

func api_refreshKey(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	request.Body = http.MaxBytesReader(response, request.Body, JSON_BODY_MAX_LENGTH)

	var p ChannelActionBody

	err := json.NewDecoder(request.Body).Decode(&p)
	if err != nil {
		response.WriteHeader(400)
		return
	}

	channel := vars["channel"]

	if !validateStreamIDString(channel) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	valid, _, _, _ := DATABASE.VerifyKey(channel, p.Key)

	if !valid {
		ReturnAPIError(response, 403, "INVALID_KEY", "Access denied")
		return
	}

	result, success := DATABASE.RefreshKey(channel)

	if !success {
		ReturnAPIError(response, 404, "CHANNEL_NOT_FOUND", "")
		return
	}

	err = DATABASE.Save()

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
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

func api_closeStream(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	request.Body = http.MaxBytesReader(response, request.Body, JSON_BODY_MAX_LENGTH)

	var p ChannelActionBody

	err := json.NewDecoder(request.Body).Decode(&p)
	if err != nil {
		response.WriteHeader(400)
		return
	}

	channel := vars["channel"]

	if !validateStreamIDString(channel) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	valid, _, _, _ := DATABASE.VerifyKey(channel, p.Key)

	if !valid {
		ReturnAPIError(response, 403, "INVALID_KEY", "Access denied")
		return
	}

	DATABASE.CloseAnyStream(channel)

	err = DATABASE.Save()

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
		return
	}

	err = StreamingCommands_CloseStream(channel, "*")

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
		return
	}

	response.WriteHeader(200)
}

func api_deleteVOD(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	request.Body = http.MaxBytesReader(response, request.Body, JSON_BODY_MAX_LENGTH)

	var p ChannelActionBody

	err := json.NewDecoder(request.Body).Decode(&p)
	if err != nil {
		response.WriteHeader(400)
		return
	}

	channel := vars["channel"]
	streamId := vars["vod"]

	if !validateStreamIDString(channel) || !validateStreamIDString(streamId) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	valid, _, _, _ := DATABASE.VerifyKey(channel, p.Key)

	if !valid {
		ReturnAPIError(response, 403, "INVALID_KEY", "Access denied")
		return
	}

	success := DATABASE.DeleteVOD(channel, streamId)

	if !success {
		ReturnAPIError(response, 404, "VOD_NOT_FOUND", "")
		return
	}

	err = DATABASE.Save()

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
		return
	}

	response.WriteHeader(200)
}

func api_deleteChannel(response http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	request.Body = http.MaxBytesReader(response, request.Body, JSON_BODY_MAX_LENGTH)

	var p ChannelActionBody

	err := json.NewDecoder(request.Body).Decode(&p)
	if err != nil {
		response.WriteHeader(400)
		return
	}

	channel := vars["channel"]

	if !validateStreamIDString(channel) {
		ReturnAPIError(response, 400, "BAD_REQUEST", "Bad request.")
		return
	}

	valid, _, _, _ := DATABASE.VerifyKey(channel, p.Key)

	if !valid {
		ReturnAPIError(response, 403, "INVALID_KEY", "Access denied")
		return
	}

	success := DATABASE.DeleteChannel(channel)

	if !success {
		ReturnAPIError(response, 404, "CHANNEL_NOT_FOUND", "")
		return
	}

	err = DATABASE.Save()

	if err != nil {
		LogError(err)

		ReturnAPIError(response, 500, "INTERNAL_ERROR", "Internal server error, Check the logs for details.")
		return
	}

	response.WriteHeader(200)
}
