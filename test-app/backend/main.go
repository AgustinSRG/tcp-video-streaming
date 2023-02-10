// Main

package main

import (
	"os"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load() // Load env vars

	InitLog()

	LogInfo("Started streaming test backend server")

	CORS_INSECURE_MODE_ENABLED = os.Getenv("CORS_INSECURE_MODE_ENABLED") == "YES"

	RunHTTPServer(os.Getenv("HTTP_PORT"), os.Getenv("BIND_ADDRESS"))
}
