package websnake

import (
	"errors"
	"net/http"
	"testing"

	"github.com/spf13/viper"
)

func TestDefaultOptions(t *testing.T) {
	opts := defaultOptions()

	if opts == nil {
		t.Fatal("expected non-nil options")
	}

	if opts.viper == nil {
		t.Error("expected non-nil viper instance")
	}

	if !opts.enforceTypeChecks {
		t.Error("expected enforceTypeChecks to be true")
	}

	if opts.allowMissingKeys {
		t.Error("expected allowMissingKeys to be false")
	}

	if opts.validationHook != nil {
		t.Error("expected validationHook to be nil")
	}

	if opts.onUpdateHook != nil {
		t.Error("expected onUpdateHook to be nil")
	}

	if opts.customConfigKeyParser != nil {
		t.Error("expected customConfigKeyParser to be nil")
	}
}

func TestEnforceTypeChecks(t *testing.T) {
	t.Run("enable type checks", func(t *testing.T) {
		opts := &Options{}
		opt := EnforceTypeChecks(true)
		opt(opts)

		if !opts.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to be true")
		}
	})

	t.Run("disable type checks", func(t *testing.T) {
		opts := &Options{enforceTypeChecks: true}
		opt := EnforceTypeChecks(false)
		opt(opts)

		if opts.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to be false")
		}
	})
}

func TestAllowMissingKeys(t *testing.T) {
	t.Run("allow missing keys", func(t *testing.T) {
		opts := &Options{}
		opt := AllowMissingKeys(true)
		opt(opts)

		if !opts.allowMissingKeys {
			t.Error("expected allowMissingKeys to be true")
		}
	})

	t.Run("disallow missing keys", func(t *testing.T) {
		opts := &Options{allowMissingKeys: true}
		opt := AllowMissingKeys(false)
		opt(opts)

		if opts.allowMissingKeys {
			t.Error("expected allowMissingKeys to be false")
		}
	})
}

func TestWithViper(t *testing.T) {
	t.Run("with custom viper instance", func(t *testing.T) {
		v := viper.New()
		v.Set("test.key", "test.value")

		opts := &Options{}
		opt := WithViper(v)
		opt(opts)

		if opts.viper == nil {
			t.Fatal("expected non-nil viper instance")
		}

		if opts.viper.GetString("test.key") != "test.value" {
			t.Error("expected custom viper instance to be set")
		}
	})

	t.Run("with nil viper returns default", func(t *testing.T) {
		opts := &Options{}
		opt := WithViper(nil)
		opt(opts)

		if opts.viper == nil {
			t.Error("expected non-nil viper instance")
		}
	})
}

func TestWithViperValues(t *testing.T) {
	t.Run("reads values from viper", func(t *testing.T) {
		v := viper.New()
		v.Set("websnake.enforce_type_checks", false)
		v.Set("websnake.allow_missing_keys", true)

		opts := &Options{
			viper:             v,
			enforceTypeChecks: true,
			allowMissingKeys:  false,
		}

		opt := WithViperValues(v)
		opt(opts)

		if opts.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to be false")
		}

		if !opts.allowMissingKeys {
			t.Error("expected allowMissingKeys to be true")
		}
	})

	t.Run("does not override if not set", func(t *testing.T) {
		v := viper.New()

		opts := &Options{
			viper:             v,
			enforceTypeChecks: true,
			allowMissingKeys:  false,
		}

		opt := WithViperValues(v)
		opt(opts)

		if !opts.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to remain true")
		}

		if opts.allowMissingKeys {
			t.Error("expected allowMissingKeys to remain false")
		}
	})

	t.Run("with nil viper uses opts.viper", func(t *testing.T) {
		v := viper.New()
		v.Set("websnake.enforce_type_checks", false)

		opts := &Options{
			viper:             v,
			enforceTypeChecks: true,
		}

		opt := WithViperValues(nil)
		opt(opts)

		if opts.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to be false")
		}
	})
}

func TestWithViperValuesPrefix(t *testing.T) {
	t.Run("reads values with prefix", func(t *testing.T) {
		v := viper.New()
		v.Set("myapp.websnake.enforce_type_checks", false)
		v.Set("myapp.websnake.allow_missing_keys", true)

		opts := &Options{
			viper:             v,
			enforceTypeChecks: true,
			allowMissingKeys:  false,
		}

		opt := WithViperValuesPrefix(v, "myapp")
		opt(opts)

		if opts.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to be false")
		}

		if !opts.allowMissingKeys {
			t.Error("expected allowMissingKeys to be true")
		}
	})

	t.Run("prefix with trailing dot", func(t *testing.T) {
		v := viper.New()
		v.Set("myapp.websnake.enforce_type_checks", false)

		opts := &Options{
			viper:             v,
			enforceTypeChecks: true,
		}

		opt := WithViperValuesPrefix(v, "myapp.")
		opt(opts)

		if opts.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to be false")
		}
	})

	t.Run("empty prefix", func(t *testing.T) {
		v := viper.New()
		v.Set("websnake.enforce_type_checks", false)

		opts := &Options{
			viper:             v,
			enforceTypeChecks: true,
		}

		opt := WithViperValuesPrefix(v, "")
		opt(opts)

		if opts.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to be false")
		}
	})

	t.Run("nil viper with opts.viper set", func(t *testing.T) {
		v := viper.New()
		v.Set("prefix.websnake.enforce_type_checks", false)

		opts := &Options{
			viper:             v,
			enforceTypeChecks: true,
		}

		opt := WithViperValuesPrefix(nil, "prefix")
		opt(opts)

		if opts.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to be false")
		}
	})

	t.Run("nil viper with nil opts.viper panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic when viper is nil")
			}
		}()

		opts := &Options{
			viper: nil,
		}

		opt := WithViperValuesPrefix(nil, "prefix")
		opt(opts)
	})
}

func TestWithValidationHook(t *testing.T) {
	t.Run("sets validation hook", func(t *testing.T) {
		called := false
		hook := func(r *http.Request, setting string, value any) error {
			called = true
			return nil
		}

		opts := &Options{}
		opt := WithValidationHook(hook)
		opt(opts)

		if opts.validationHook == nil {
			t.Fatal("expected validationHook to be set")
		}

		// Test that the hook works
		opts.validationHook(nil, "test", "value")
		if !called {
			t.Error("expected hook to be called")
		}
	})

	t.Run("hook returns error", func(t *testing.T) {
		expectedErr := errors.New("validation error")
		hook := func(r *http.Request, setting string, value any) error {
			return expectedErr
		}

		opts := &Options{}
		opt := WithValidationHook(hook)
		opt(opts)

		err := opts.validationHook(nil, "test", "value")
		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("nil hook", func(t *testing.T) {
		opts := &Options{}
		opt := WithValidationHook(nil)
		opt(opts)

		if opts.validationHook != nil {
			t.Error("expected validationHook to be nil")
		}
	})
}

func TestWithOnUpdateHook(t *testing.T) {
	t.Run("sets onUpdate hook", func(t *testing.T) {
		called := false
		var capturedSetting string
		var capturedValue any

		hook := func(setting string, value any) {
			called = true
			capturedSetting = setting
			capturedValue = value
		}

		opts := &Options{}
		opt := WithOnUpdateHook(hook)
		opt(opts)

		if opts.onUpdateHook == nil {
			t.Fatal("expected onUpdateHook to be set")
		}

		// Test that the hook works
		opts.onUpdateHook("test.key", "test.value")
		if !called {
			t.Error("expected hook to be called")
		}
		if capturedSetting != "test.key" {
			t.Errorf("expected setting 'test.key', got '%s'", capturedSetting)
		}
		if capturedValue != "test.value" {
			t.Errorf("expected value 'test.value', got '%v'", capturedValue)
		}
	})

	t.Run("nil hook", func(t *testing.T) {
		opts := &Options{}
		opt := WithOnUpdateHook(nil)
		opt(opts)

		if opts.onUpdateHook != nil {
			t.Error("expected onUpdateHook to be nil")
		}
	})
}

func TestWithCustomConfigKeyParser(t *testing.T) {
	t.Run("sets custom parser", func(t *testing.T) {
		called := false
		parser := func(r *http.Request) ([]string, error) {
			called = true
			return []string{"key1", "key2"}, nil
		}

		opts := &Options{}
		opt := WithCustomConfigKeyParser(parser)
		opt(opts)

		if opts.customConfigKeyParser == nil {
			t.Fatal("expected customConfigKeyParser to be set")
		}

		// Test that the parser works
		keys, err := opts.customConfigKeyParser(nil)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !called {
			t.Error("expected parser to be called")
		}
		if len(keys) != 2 {
			t.Errorf("expected 2 keys, got %d", len(keys))
		}
	})

	t.Run("parser returns error", func(t *testing.T) {
		expectedErr := errors.New("parse error")
		parser := func(r *http.Request) ([]string, error) {
			return nil, expectedErr
		}

		opts := &Options{}
		opt := WithCustomConfigKeyParser(parser)
		opt(opts)

		_, err := opts.customConfigKeyParser(nil)
		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("nil parser", func(t *testing.T) {
		opts := &Options{}
		opt := WithCustomConfigKeyParser(nil)
		opt(opts)

		if opts.customConfigKeyParser != nil {
			t.Error("expected customConfigKeyParser to be nil")
		}
	})
}

func TestOptionChaining(t *testing.T) {
	t.Run("multiple options applied in order", func(t *testing.T) {
		v := viper.New()
		v.Set("test", "value")

		validationCalled := false
		updateCalled := false

		ws := NewWebSnake(
			WithViper(v),
			EnforceTypeChecks(false),
			AllowMissingKeys(true),
			WithValidationHook(func(r *http.Request, setting string, value any) error {
				validationCalled = true
				return nil
			}),
			WithOnUpdateHook(func(setting string, value any) {
				updateCalled = true
			}),
		)

		if ws.options.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to be false")
		}
		if !ws.options.allowMissingKeys {
			t.Error("expected allowMissingKeys to be true")
		}
		if ws.options.viper.GetString("test") != "value" {
			t.Error("expected custom viper to be set")
		}
		if ws.options.validationHook == nil {
			t.Error("expected validationHook to be set")
		}
		if ws.options.onUpdateHook == nil {
			t.Error("expected onUpdateHook to be set")
		}

		// Verify hooks work
		ws.options.validationHook(nil, "test", "value")
		ws.options.onUpdateHook("test", "value")

		if !validationCalled {
			t.Error("expected validation hook to be called")
		}
		if !updateCalled {
			t.Error("expected update hook to be called")
		}
	})
}
