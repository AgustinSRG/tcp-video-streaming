// Key verification callback

package main

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"os"
	"strings"
)

func checkKeyVerificationAuth(auth string) bool {
	authMethod := strings.ToUpper(os.Getenv("KEY_VERIFICATION_AUTH"))
	authorization := ""

	switch authMethod {
	case "BASIC":
		user := os.Getenv("KEY_VERIFICATION_AUTH_USER")
		password := os.Getenv("KEY_VERIFICATION_PASSWORD")
		authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
	case "BEARER":
		token := os.Getenv("KEY_VERIFICATION_AUTH_TOKEN")
		authorization = "Bearer " + token
	case "CUSTOM":
		authorization = os.Getenv("KEY_VERIFICATION_AUTH_CUSTOM")
	}

	return subtle.ConstantTimeCompare([]byte(authorization), []byte(auth)) == 1
}

func callback_keyVerification(response http.ResponseWriter, request *http.Request) {
	if !checkKeyVerificationAuth(request.Header.Get("Authentication")) {
		ReturnAPIError(response, 401, "UNAUTHORIZED", "Invalid authorization.")
		return
	}

	channel := request.Header.Get("x-streaming-channel")
	key := request.Header.Get("x-streaming-key")
	ip := request.Header.Get("x-user-ip")

	LogDebug("Incoming key verification. Channel=" + channel + ", Key=" + key + ", User IP=" + ip)

	keyValid, record, previews, resolutions := DATABASE.VerifyKey(channel, key)

	if !keyValid {
		ReturnAPIError(response, 403, "INVALID_KEY", "Invalid streaming key")
		return
	}

	if record {
		response.Header().Add("x-record", "true")
	} else {
		response.Header().Add("x-record", "false")
	}
	response.Header().Add("x-previews", previews)
	response.Header().Add("x-resolutions", resolutions)
	response.WriteHeader(200)
	response.Write([]byte("OK"))
}
