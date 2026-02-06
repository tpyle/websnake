// Package main demonstrates a simple websnake HTTP server for dynamic configuration management.
// This example creates HTTP endpoints for updating and retrieving configuration settings.
//
// Start the server with: go run main.go
//
// Example usage:
//
// Update a setting:
//
//	curl -X POST http://localhost:8080/config/update \
//	  -H "Content-Type: application/json" \
//	  -d '{"setting": "app.port", "value": 8080}'
//
// Retrieve a setting:
//
//	curl http://localhost:8080/config/get/app.port
package main

import (
	"net/http"

	"github.com/tpyle/websnake"
)

func main() {
	// Create a new WebSnake instance that allows creating new configuration keys
	ws := websnake.NewWebSnake(websnake.AllowMissingKeys(true))

	// Register HTTP handlers
	http.HandleFunc("/config/update", ws.HandleConfigUpdate)
	http.HandleFunc("/config/get/", ws.HandleConfigGet)

	// Start the HTTP server on port 8080
	http.ListenAndServe(":8080", nil)
}
