# WebSnake

WebSnake is a Go library that provides HTTP handlers for dynamically managing application configuration settings. It integrates seamlessly with [Viper](https://github.com/spf13/viper) to enable runtime configuration updates and retrievals via HTTP endpoints.

## Features

- 🔄 **Dynamic Configuration Updates** - Modify configuration settings at runtime via HTTP requests
- 🔍 **Configuration Retrieval** - Query configuration values through a RESTful API
- ✅ **Type Safety** - Optional type checking to prevent configuration type mismatches
- 🔒 **Validation Hooks** - Custom validation logic before accepting updates
- 📢 **Update Notifications** - Callbacks triggered after successful configuration changes
- 🎯 **Custom Key Parsing** - Flexible request parsing for batch operations or custom URL patterns
- 🔑 **Missing Key Control** - Choose whether to allow creating new configuration keys
- 🧪 **Well Tested** - 100% test coverage

## Installation

```bash
go get -u github.com/tpyle/websnake
```

## Quick Start

### Basic Usage

```go
package main

import (
    "net/http"
    "github.com/tpyle/websnake"
)

func main() {
    // Create a new WebSnake instance with default settings
    ws := websnake.NewWebSnake()

    // Register HTTP handlers
    http.HandleFunc("/config/update", ws.HandleConfigUpdate)
    http.HandleFunc("/config/get/", ws.HandleConfigGet)

    // Start the server
    http.ListenAndServe(":8080", nil)
}
```

### Update a Configuration Setting

```bash
curl -X POST http://localhost:8080/config/update \
  -H "Content-Type: application/json" \
  -d '{"setting": "app.port", "value": 8080}'
```

Response:
```json
{
  "setting": "app.port",
  "value": 8080
}
```

### Retrieve a Configuration Setting

```bash
curl http://localhost:8080/config/get/app.port
```

Response:
```json
[{
  "setting": "app.port",
  "value": 8080
}]
```

## Configuration Options

WebSnake uses the functional options pattern for flexible configuration:

### EnforceTypeChecks

Control whether type checking is enforced when updating settings.

```go
ws := websnake.NewWebSnake(
    websnake.EnforceTypeChecks(false), // Disable type checking
)
```

**Default:** `true` (type checking enabled)

### AllowMissingKeys

Allow updates and retrievals of configuration keys that don't exist.

```go
ws := websnake.NewWebSnake(
    websnake.AllowMissingKeys(true), // Allow creating new keys
)
```

**Default:** `false` (only existing keys can be accessed)

### WithViper

Use a custom Viper instance instead of the global one.

```go
v := viper.New()
v.SetDefault("app.port", 8080)
v.SetDefault("app.name", "MyApp")

ws := websnake.NewWebSnake(
    websnake.WithViper(v),
)
```

### WithValidationHook

Add custom validation logic before accepting configuration updates.

```go
validator := func(r *http.Request, setting string, value any) error {
    if setting == "app.port" {
        if port, ok := value.(float64); ok && (port < 1024 || port > 65535) {
            return fmt.Errorf("port must be between 1024 and 65535")
        }
    }
    return nil
}

ws := websnake.NewWebSnake(
    websnake.WithValidationHook(validator),
)
```

### WithOnUpdateHook

Execute custom logic after successful configuration updates.

```go
onUpdate := func(setting string, value any) {
    log.Printf("Configuration updated: %s = %v", setting, value)

    // Trigger side effects
    if setting == "app.log_level" {
        updateLogLevel(value.(string))
    }
}

ws := websnake.NewWebSnake(
    websnake.WithOnUpdateHook(onUpdate),
)
```

### WithCustomConfigKeyParser

Implement custom URL parsing to retrieve multiple settings or use custom routing.

```go
parser := func(r *http.Request) ([]string, error) {
    // Get multiple keys from query parameter: ?keys=key1,key2,key3
    keysParam := r.URL.Query().Get("keys")
    if keysParam == "" {
        return nil, fmt.Errorf("keys parameter required")
    }
    return strings.Split(keysParam, ","), nil
}

ws := websnake.NewWebSnake(
    websnake.WithCustomConfigKeyParser(parser),
)
```

### WithViperValues / WithViperValuesPrefix

Configure WebSnake options using values from a Viper configuration.

```go
v := viper.New()
v.Set("websnake.enforce_type_checks", false)
v.Set("websnake.allow_missing_keys", true)

ws := websnake.NewWebSnake(
    websnake.WithViperValues(v),
)
```

With a custom prefix:

```go
v := viper.New()
v.Set("myapp.websnake.enforce_type_checks", false)

ws := websnake.NewWebSnake(
    websnake.WithViperValuesPrefix(v, "myapp"),
)
```

## Advanced Example

Combining multiple options for a production-ready configuration service:

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "github.com/spf13/viper"
    "github.com/tpyle/websnake"
)

func main() {
    // Initialize Viper with default configuration
    v := viper.New()
    v.SetDefault("app.port", 8080)
    v.SetDefault("app.name", "MyApp")
    v.SetDefault("app.log_level", "info")

    // Create WebSnake with comprehensive configuration
    ws := websnake.NewWebSnake(
        websnake.WithViper(v),
        websnake.AllowMissingKeys(false), // Only allow updates to existing keys
        websnake.EnforceTypeChecks(true), // Prevent type mismatches
        websnake.WithValidationHook(validateConfig),
        websnake.WithOnUpdateHook(onConfigUpdate),
    )

    // Register handlers
    http.HandleFunc("/config/update", ws.HandleConfigUpdate)
    http.HandleFunc("/config/get/", ws.HandleConfigGet)

    log.Printf("Starting configuration service on port %d", v.GetInt("app.port"))
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", v.GetInt("app.port")), nil))
}

func validateConfig(r *http.Request, setting string, value any) error {
    switch setting {
    case "app.port":
        if port, ok := value.(float64); ok {
            if port < 1024 || port > 65535 {
                return fmt.Errorf("port must be between 1024 and 65535")
            }
        }
    case "app.log_level":
        if level, ok := value.(string); ok {
            validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
            if !validLevels[level] {
                return fmt.Errorf("log_level must be debug, info, warn, or error")
            }
        }
    }
    return nil
}

func onConfigUpdate(setting string, value any) {
    log.Printf("Configuration changed: %s = %v", setting, value)

    // Trigger necessary side effects
    switch setting {
    case "app.log_level":
        // Update logger configuration
        log.Printf("Updating log level to: %v", value)
    case "app.port":
        // Note: Changing port requires server restart
        log.Printf("Port changed to %v - restart required", value)
    }
}
```

## API Reference

### HTTP Endpoints

#### POST /config/update

Update a configuration setting.

**Headers:**
- `Content-Type: application/json` (required)

**Request Body:**
```json
{
  "setting": "key.name",
  "value": "any-value"
}
```

**Response Codes:**
- `200 OK` - Setting updated successfully
- `400 Bad Request` - Invalid JSON, type mismatch, or validation failure
- `404 Not Found` - Setting not found (when `allowMissingKeys` is `false`)
- `415 Unsupported Media Type` - Content-Type header is not application/json

#### GET /config/get/{key}

Retrieve a configuration setting.

**Response Codes:**
- `200 OK` - Setting retrieved successfully
- `400 Bad Request` - Custom parser error
- `404 Not Found` - Setting not found

## Testing

WebSnake has 100% test coverage. Run the tests with:

```bash
go test -v -cover
```

Run tests with detailed coverage report:

```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Examples

See the [examples](examples/) directory for complete working examples:

- [simple](examples/simple/) - Basic configuration server with minimal setup

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.

## License

See [LICENSE](LICENSE) file for details.

## Related Projects

- [Viper](https://github.com/spf13/viper) - Go configuration with fangs
