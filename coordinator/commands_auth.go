// Commands authentication

package main

import (
	"crypto/subtle"
	"encoding/base64"
	"os"
	"strings"
)

// Checks command authentication
// authHeader - Provided authentication header
// Returns true only if the authentication is valid
func CheckCommandAuthentication(authHeader string) bool {
	spaceIndex := strings.Index(authHeader, " ")

	if spaceIndex == -1 || spaceIndex == len(authHeader)-1 {
		return false
	}

	authHeaderMode := strings.ToUpper(authHeader[:spaceIndex])
	authHeaderValue := authHeader[spaceIndex+1:]

	authMode := strings.ToUpper(os.Getenv("COMMANDS_API_AUTH"))

	if authMode != authHeaderMode {
		return false
	}

	switch authMode {
	case "BASIC":
		userExpected := os.Getenv("COMMANDS_API_AUTH_USER")
		passwordExpected := os.Getenv("COMMANDS_API_AUTH_TOKEN")

		rawDecodedText, err := base64.StdEncoding.DecodeString(authHeaderValue)

		if err != nil {
			return false
		}

		return subtle.ConstantTimeCompare([]byte(rawDecodedText), []byte(userExpected+":"+passwordExpected)) == 1
	case "BEARER":
		tokenExpected := os.Getenv("COMMANDS_API_AUTH_TOKEN")
		return subtle.ConstantTimeCompare([]byte(authHeaderValue), []byte(tokenExpected)) == 1
	default:
		return false // No authentication method set
	}
}
