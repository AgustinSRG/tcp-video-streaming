// Key validation system

package main

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"
)

// Validates a stream key
// channel - The channel
// key - The stream key
// userIP - IP of the publisher
// Returns:
//   valid - True only if the key is valid
//   resolutionList - List of allowed resolutions
//   record - True if recording is enabled
//   previewsConfig - Previews configuration
func ValidateStreamKey(channel string, key string, userIP string) (valid bool, resolutionList ResolutionList, record bool, previewsConfig PreviewsConfiguration) {
	verificationURL := os.Getenv("KEY_VERIFICATION_URL")

	if verificationURL == "" {
		LogWarning("Key was considered valid by default, since KEY_VERIFICATION_URL is missing")
		return true, ResolutionList{hasOriginal: true, resolutions: make([]Resolution, 0)}, false, PreviewsConfiguration{enabled: false}
	}

	authorization := ""

	authMethod := strings.ToUpper(os.Getenv("KEY_VERIFICATION_AUTH"))

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

	client := &http.Client{}

	req, e := http.NewRequest("POST", verificationURL, nil)

	if e != nil {
		LogError(e)
		return false, ResolutionList{}, false, PreviewsConfiguration{}
	}

	req.Header.Set("x-streaming-channel", channel)
	req.Header.Set("x-streaming-key", key)
	req.Header.Set("x-user-ip", userIP)

	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}

	LogDebug("Validating stream key for channel: " + channel + " / POST: " + verificationURL)

	res, e := client.Do(req)

	if e != nil {
		LogError(e)
		return false, ResolutionList{}, false, PreviewsConfiguration{}
	}

	if res.StatusCode == 200 {
		return true, DecodeResolutionsList(res.Header.Get("x-resolutions")), strings.ToLower(res.Header.Get("x-record")) == "true", DecodePreviewsConfiguration(res.Header.Get("x-previews"))
	} else {
		return false, ResolutionList{}, false, PreviewsConfiguration{}
	}
}
