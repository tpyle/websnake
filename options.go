package websnake

import (
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

// Option is a functional option for configuring a WebSnake instance.
// Options allow flexible configuration of behavior such as type checking,
// validation hooks, and custom key parsing.
type Option func(*Options)

type Options struct {
	viper *viper.Viper

	enforceTypeChecks bool
	allowMissingKeys  bool

	validationHook func(r *http.Request, setting string, value any) error
	onUpdateHook   func(setting string, value any)

	customConfigKeyParser func(r *http.Request) ([]string, error)
}

func defaultOptions() *Options {
	return &Options{
		viper:                 viper.GetViper(),
		enforceTypeChecks:     true,
		allowMissingKeys:      false,
		validationHook:        nil,
		onUpdateHook:          nil,
		customConfigKeyParser: nil,
	}
}

// EnforceTypeChecks configures whether type checking is enforced when updating settings.
// When enabled (default), updates must match the existing value's type.
// When disabled, settings can be changed to any type.
//
// Example:
//
//	ws := websnake.NewWebSnake(websnake.EnforceTypeChecks(false))
func EnforceTypeChecks(enforce bool) Option {
	return func(opts *Options) {
		opts.enforceTypeChecks = enforce
	}
}

// AllowMissingKeys configures whether updates and retrievals can reference keys
// that don't exist in the Viper configuration.
// When false (default), only existing keys can be accessed or modified.
// When true, new keys can be created via updates.
//
// Example:
//
//	ws := websnake.NewWebSnake(websnake.AllowMissingKeys(true))
func AllowMissingKeys(allow bool) Option {
	return func(opts *Options) {
		opts.allowMissingKeys = allow
	}
}

// WithViper configures WebSnake to use a specific Viper instance instead of the global one.
// If nil is passed, it falls back to the global Viper instance (viper.GetViper()).
//
// Example:
//
//	v := viper.New()
//	v.SetDefault("app.port", 8080)
//	ws := websnake.NewWebSnake(websnake.WithViper(v))
func WithViper(v *viper.Viper) Option {
	return func(opts *Options) {
		if v == nil {
			v = viper.GetViper()
		}
		opts.viper = v
	}
}

// WithViperValues configures WebSnake options using values from a Viper instance.
// It reads the following settings:
//   - websnake.enforce_type_checks (bool)
//   - websnake.allow_missing_keys (bool)
//
// This is equivalent to calling WithViperValuesPrefix(v, "").
//
// Example:
//
//	v := viper.New()
//	v.Set("websnake.enforce_type_checks", false)
//	ws := websnake.NewWebSnake(websnake.WithViperValues(v))
func WithViperValues(v *viper.Viper) Option {
	return WithViperValuesPrefix(v, "")
}

// WithViperValuesPrefix configures WebSnake options using values from a Viper instance
// with a custom prefix. The prefix is prepended to "websnake." settings.
// If the prefix doesn't end with ".", one is automatically added.
//
// It reads the following settings:
//   - {prefix}.websnake.enforce_type_checks (bool)
//   - {prefix}.websnake.allow_missing_keys (bool)
//
// If v is nil, it uses the Viper instance from opts.viper (panics if both are nil).
//
// Example:
//
//	v := viper.New()
//	v.Set("myapp.websnake.enforce_type_checks", false)
//	ws := websnake.NewWebSnake(websnake.WithViperValuesPrefix(v, "myapp"))
func WithViperValuesPrefix(v *viper.Viper, prefix string) Option {
	return func(opts *Options) {
		if v == nil {
			if opts.viper != nil {
				v = opts.viper
			} else {
				// This should never happen, but just in case
				panic("viper instance is nil, cannot read viper values")
			}
		}

		if prefix != "" && !strings.HasSuffix(prefix, ".") {
			prefix += "."
		}

		if v.IsSet(prefix + "websnake.enforce_type_checks") {
			opts.enforceTypeChecks = v.GetBool(prefix + "websnake.enforce_type_checks")
		}
		if v.IsSet(prefix + "websnake.allow_missing_keys") {
			opts.allowMissingKeys = v.GetBool(prefix + "websnake.allow_missing_keys")
		}
	}
}

// WithValidationHook configures a custom validation function that is called before
// any configuration update. The hook receives the HTTP request, setting name, and new value.
// If the hook returns an error, the update is rejected with a 400 Bad Request response.
//
// The validation hook can be used to:
//   - Enforce business rules on configuration values
//   - Validate value ranges or formats
//   - Implement authorization checks
//
// Example:
//
//	validator := func(r *http.Request, setting string, value any) error {
//		if setting == "app.port" {
//			if port, ok := value.(float64); ok && (port < 1024 || port > 65535) {
//				return fmt.Errorf("port must be between 1024 and 65535")
//			}
//		}
//		return nil
//	}
//	ws := websnake.NewWebSnake(websnake.WithValidationHook(validator))
func WithValidationHook(hook func(r *http.Request, setting string, value any) error) Option {
	return func(opts *Options) {
		opts.validationHook = hook
	}
}

// WithOnUpdateHook configures a callback function that is called after a configuration
// setting has been successfully updated. The hook receives the setting name and new value.
//
// The update hook can be used to:
//   - Log configuration changes
//   - Trigger side effects (reload services, invalidate caches, etc.)
//   - Notify other systems of configuration changes
//
// Example:
//
//	onUpdate := func(setting string, value any) {
//		log.Printf("Configuration updated: %s = %v", setting, value)
//		if setting == "app.log_level" {
//			updateLogLevel(value.(string))
//		}
//	}
//	ws := websnake.NewWebSnake(websnake.WithOnUpdateHook(onUpdate))
func WithOnUpdateHook(hook func(setting string, value any)) Option {
	return func(opts *Options) {
		opts.onUpdateHook = hook
	}
}

// WithCustomConfigKeyParser configures a custom function for extracting configuration
// key names from HTTP requests. This allows retrieving multiple settings in a single
// request or using custom URL patterns.
//
// By default, HandleConfigGet extracts the last segment of the URL path as the setting name.
// With a custom parser, you can:
//   - Extract multiple keys from query parameters
//   - Parse keys from request headers
//   - Implement custom URL routing logic
//
// The parser should return a slice of setting names to retrieve, or an error if parsing fails.
//
// Example:
//
//	parser := func(r *http.Request) ([]string, error) {
//		// Get multiple keys from query parameter: ?keys=key1,key2,key3
//		keysParam := r.URL.Query().Get("keys")
//		if keysParam == "" {
//			return nil, fmt.Errorf("keys parameter required")
//		}
//		return strings.Split(keysParam, ","), nil
//	}
//	ws := websnake.NewWebSnake(websnake.WithCustomConfigKeyParser(parser))
func WithCustomConfigKeyParser(parser func(r *http.Request) ([]string, error)) Option {
	return func(opts *Options) {
		opts.customConfigKeyParser = parser
	}
}
