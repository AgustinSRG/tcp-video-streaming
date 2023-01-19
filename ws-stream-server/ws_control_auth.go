// Websocket authentication

package main

import (
	"os"

	"github.com/golang-jwt/jwt"
)

func MakeWebsocketAuthenticationToken() string {
	secret := os.Getenv("CONTROL_SECRET")

	if secret == "" {
		return ""
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "wss-control",
	})

	tokenBase64, e := token.SignedString([]byte(secret))

	if e != nil {
		LogError(e)
		return ""
	}

	return tokenBase64
}
