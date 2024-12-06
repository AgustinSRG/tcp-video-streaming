// Server authentication utils

package main

import (
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

const (
	RTMP_AUTH_SUBJECT = "rtmp-control"
	WSS_AUTH_SUBJECT  = "wss-control"
	HLS_AUTH_SUBJECT  = "hls-control"
)

// Validates authentication token
// token - Auth token to validate
// requiredSubject - Subject required by the token
// Returns true only if valid
func ValidateAuthenticationToken(token string, requiredSubject string) bool {
	secret := os.Getenv("CONTROL_SECRET")

	if secret == "" {
		return true // If no secret set, any token becomes valid
	}

	if token == "" {
		return false
	}

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Check the algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		// Provide signing key
		return []byte(secret), nil
	})

	if err != nil {
		return false // Invalid token
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)

	if !ok || !parsedToken.Valid {
		return false // Invalid token
	}

	if claims["sub"] == nil || claims["sub"].(string) != requiredSubject {
		return false // Invalid subject
	}

	return true // Valid
}
