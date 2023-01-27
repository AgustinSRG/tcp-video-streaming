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
// Returns true only if the key is valid
func ValidateStreamKey(channel string, key string, userIP string) bool {
	verificationURL := os.Getenv("KEY_VERIFICATION_URL")

	if verificationURL == "" {
		LogWarning("Key was considered valid by default, since KEY_VERIFICATION_URL is missing")
		return true
	}

	authorization := ""

	authMethod := strings.ToUpper(os.Getenv("KEY_VERIFICATION_AUTH"))

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

	client := &http.Client{}

	req, e := http.NewRequest("POST", verificationURL, nil)

	if e != nil {
		LogError(e)
		return false
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
		return false
	}

	return res.StatusCode == 200
}
