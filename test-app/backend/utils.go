// Utilities

package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

const JSON_BODY_MAX_LENGTH = 5 * 1024 * 1024

// Returns API standard JSON response
// response - HTTP response handler
// request - HTTP request handler
// result - JSON result
func ReturnAPI_JSON(response http.ResponseWriter, request *http.Request, result []byte) {
	hasher := sha256.New()
	hasher.Write(result)
	hash := hasher.Sum(nil)
	etag := hex.EncodeToString(hash)

	response.Header().Set("ETag", etag)
	response.Header().Add("Cache-Control", "no-cache")

	if request.Header.Get("If-None-Match") == etag {
		response.WriteHeader(304)
	} else {
		response.Header().Add("Content-Type", "application/json")
		response.WriteHeader(200)

		response.Write(result)
	}
}

// API standard error response
type APIErrorResponse struct {
	Code    string `json:"code"`    // Error code
	Message string `json:"message"` // Error message
}

// Returns API standard error message
// response - HTTP response handler
// request - HTTP request handler
// status - HTTP status
// code - Error code
// message - Error message
func ReturnAPIError(response http.ResponseWriter, status int, code string, message string) {
	var m APIErrorResponse

	m.Code = code
	m.Message = message

	jsonRes, err := json.Marshal(m)

	if err != nil {
		LogError(err)
		response.Header().Add("Cache-Control", "no-cache")
		response.WriteHeader(500)
		return
	}

	response.Header().Add("Content-Type", "application/json")
	response.Header().Add("Cache-Control", "no-cache")
	response.WriteHeader(status)
	response.Write(jsonRes)
}

// Gets client IP address
// request - HTTP request
func GetClientIP(request *http.Request) string {
	ip, _, _ := net.SplitHostPort(request.RemoteAddr)

	if os.Getenv("USING_PROXY") == "YES" {
		forwardedFor := request.Header.Get("X-Forwarded-For")

		if forwardedFor != "" {
			return forwardedFor
		} else {
			return ip
		}
	} else {
		return ip
	}
}

// Validates stream ID
// str - Stream ID
// Returns true only if valid
func validateStreamIDString(str string) bool {
	var ID_MAX_LENGTH = 128
	var idCustomMaxLength = os.Getenv("ID_MAX_LENGTH")

	if idCustomMaxLength != "" {
		var e error
		ID_MAX_LENGTH, e = strconv.Atoi(idCustomMaxLength)
		if e != nil {
			ID_MAX_LENGTH = 128
		}
	}

	if len(str) > ID_MAX_LENGTH {
		return false
	}

	m, e := regexp.MatchString("^[A-Za-z0-9\\_\\-]+$", str)

	if e != nil {
		return false
	}

	return m
}

func generateRandomKey() string {
	keyBytes := make([]byte, 32)
	rand.Read(keyBytes)
	return hex.EncodeToString(keyBytes)
}
