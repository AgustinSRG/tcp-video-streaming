// Main

package main

import "github.com/joho/godotenv"

func main() {
	godotenv.Load() // Load env vars

	InitLog()

	LogInfo("Started streaming test backend server")

}
