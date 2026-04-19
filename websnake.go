// Package websnake provides HTTP handlers for dynamically managing application configuration
// settings via HTTP requests. It integrates with Viper for configuration management and
// supports features like type checking, validation hooks, and custom key parsing.
//
// Basic usage:
//
//	ws := websnake.NewWebSnake()
//	http.HandleFunc("/config/update", ws.HandleConfigUpdate)
//	http.HandleFunc("/config/get/", ws.HandleConfigGet)
//	http.ListenAndServe(":8080", nil)
//
// With options:
//
//	ws := websnake.NewWebSnake(
//		websnake.AllowMissingKeys(true),
//		websnake.EnforceTypeChecks(false),
//	)
package websnake

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// WebSnake provides HTTP handlers for managing configuration settings dynamically.
// It wraps a Viper instance and provides configurable validation and update hooks.
type WebSnake struct {
	options *Options
}

// NewWebSnake creates a new WebSnake instance with the provided options.
// If no options are provided, it uses sensible defaults:
//   - Type checking is enabled
//   - Missing keys are not allowed
//   - Uses the global Viper instance
//
// Example:
//
//	ws := websnake.NewWebSnake(
//		websnake.WithViper(myViper),
//		websnake.AllowMissingKeys(true),
//	)
func NewWebSnake(opts ...Option) *WebSnake {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	return &WebSnake{
		options: options,
	}
}

// ConfigUpdateRequest represents the JSON payload for updating a configuration setting.
// The Setting field specifies which configuration key to update, and Value contains
// the new value to set.
type ConfigUpdateRequest struct {
	Setting string `json:"setting"`
	Value   any    `json:"value"`
}

// ConfigGetResponse represents the JSON response when retrieving a configuration setting.
// It contains the setting name and its current value.
type ConfigGetResponse struct {
	Setting string `json:"setting"`
	Value   any    `json:"value"`
}

// HandleConfigUpdate is an HTTP handler for updating configuration settings.
// It expects a JSON payload with "setting" and "value" fields.
// The request must have Content-Type: application/json header.
//
// The handler performs the following checks:
//   - Validates Content-Type header is application/json
//   - Validates the JSON request body
//   - Checks if the setting exists (if allowMissingKeys is false)
//   - Enforces type checking (if enforceTypeChecks is true)
//   - Runs the validation hook (if configured)
//   - Updates the setting in Viper
//   - Calls the onUpdate hook (if configured)
//
// Returns:
//   - 200 OK with the updated setting on success
//   - 400 Bad Request on invalid JSON, type mismatch, or validation failure
//   - 404 Not Found if the setting doesn't exist and allowMissingKeys is false
//   - 415 Unsupported Media Type if Content-Type is not application/json
//
// Example request:
//
//	POST /config/update
//	Content-Type: application/json
//
//	{
//	  "setting": "app.port",
//	  "value": 8080
//	}
func (ws *WebSnake) HandleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var req ConfigUpdateRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if !ws.options.allowMissingKeys && !ws.options.viper.IsSet(req.Setting) {
		http.Error(w, "Setting not found", http.StatusNotFound)
		return
	}

	if ws.options.enforceTypeChecks {
		existingValue := ws.options.viper.Get(req.Setting)
		if existingValue != nil && reflect.TypeOf(existingValue) != reflect.TypeOf(req.Value) {
			http.Error(w, fmt.Sprintf("Type mismatch for setting. Expected %s, got %s", reflect.TypeOf(existingValue).String(), reflect.TypeOf(req.Value).String()), http.StatusBadRequest)
			return
		}
	}

	if ws.options.validationHook != nil {
		if err := ws.options.validationHook(r, req.Setting, req.Value); err != nil {
			http.Error(w, "Validation failed: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	ws.options.viper.Set(req.Setting, req.Value)
	if ws.options.onUpdateHook != nil {
		ws.options.onUpdateHook(req.Setting, req.Value)
	}

	res := ConfigGetResponse{
		Setting: req.Setting,
		Value:   req.Value,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

// HandleConfigGet is an HTTP handler for retrieving configuration settings.
// By default, it extracts the setting name from the last segment of the URL path.
// If a custom config key parser is configured, it uses that instead to potentially
// retrieve multiple settings at once.
//
// The handler performs the following:
//   - Parses the setting name(s) from the request
//   - Checks if each setting exists (if allowMissingKeys is false)
//   - Retrieves the value(s) from Viper
//   - Returns the result(s) as JSON
//
// Returns:
//   - 200 OK with an array of setting/value pairs on success
//   - 400 Bad Request if custom parser fails
//   - 404 Not Found if any setting doesn't exist
//
// Example request:
//
//	GET /config/get/app.port
//
// Example response:
//
//	[{
//	  "setting": "app.port",
//	  "value": 8080
//	}]
func (ws *WebSnake) HandleConfigGet(w http.ResponseWriter, r *http.Request) {
	var settings []string
	var err error

	if ws.options.customConfigKeyParser != nil {
		settings, err = ws.options.customConfigKeyParser(r)
		if err != nil {
			http.Error(w, "Failed to parse config key: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		parts := strings.Split(r.URL.Path, "/")

		setting := parts[len(parts)-1]
		if setting == "" {
			settings = ws.options.viper.AllKeys()
		} else {
			settings = []string{parts[len(parts)-1]}
		}
	}

	res := make([]ConfigGetResponse, 0, len(settings))

	for _, setting := range settings {
		if !ws.options.allowMissingKeys && !ws.options.viper.IsSet(setting) {
			http.Error(w, "Setting not found: "+setting, http.StatusNotFound)
			return
		}

		value := ws.options.viper.Get(setting)
		if value == nil {
			http.Error(w, "Setting not found: "+setting, http.StatusNotFound)
			return
		}

		res = append(res, ConfigGetResponse{
			Setting: setting,
			Value:   value,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func (ws *WebSnake) Handler() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("PUT /config/update", ws.HandleConfigUpdate)
	mux.HandleFunc("GET /config/get/", ws.HandleConfigGet)

	return mux
}
