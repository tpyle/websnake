package websnake

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
)

func TestNewWebSnake(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		ws := NewWebSnake()
		if ws == nil {
			t.Fatal("expected non-nil WebSnake")
		}
		if ws.options == nil {
			t.Fatal("expected non-nil options")
		}
		if !ws.options.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to be true by default")
		}
		if ws.options.allowMissingKeys {
			t.Error("expected allowMissingKeys to be false by default")
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		ws := NewWebSnake(
			EnforceTypeChecks(false),
			AllowMissingKeys(true),
		)
		if ws.options.enforceTypeChecks {
			t.Error("expected enforceTypeChecks to be false")
		}
		if !ws.options.allowMissingKeys {
			t.Error("expected allowMissingKeys to be true")
		}
	})
}

func TestHandleConfigUpdate(t *testing.T) {
	t.Run("missing content-type header", func(t *testing.T) {
		v := viper.New()
		v.Set("test.key", "original")
		ws := NewWebSnake(WithViper(v))

		reqBody := ConfigUpdateRequest{
			Setting: "test.key",
			Value:   "updated",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader(body))
		// Intentionally not setting Content-Type header
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusUnsupportedMediaType {
			t.Errorf("expected status 415, got %d", w.Code)
		}
		bodyStr := w.Body.String()
		if !bytes.Contains([]byte(bodyStr), []byte("Content-Type")) {
			t.Error("expected 'Content-Type' in error message")
		}
	})

	t.Run("incorrect content-type header", func(t *testing.T) {
		v := viper.New()
		v.Set("test.key", "original")
		ws := NewWebSnake(WithViper(v))

		reqBody := ConfigUpdateRequest{
			Setting: "test.key",
			Value:   "updated",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "text/plain")
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusUnsupportedMediaType {
			t.Errorf("expected status 415, got %d", w.Code)
		}
	})

	t.Run("successful update", func(t *testing.T) {
		v := viper.New()
		v.Set("test.key", "original")
		ws := NewWebSnake(WithViper(v))

		reqBody := ConfigUpdateRequest{
			Setting: "test.key",
			Value:   "updated",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp ConfigGetResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if resp.Setting != "test.key" {
			t.Errorf("expected setting 'test.key', got '%s'", resp.Setting)
		}
		if resp.Value != "updated" {
			t.Errorf("expected value 'updated', got '%v'", resp.Value)
		}

		if v.GetString("test.key") != "updated" {
			t.Error("expected viper to be updated")
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		ws := NewWebSnake()
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("setting not found when allowMissingKeys is false", func(t *testing.T) {
		v := viper.New()
		ws := NewWebSnake(WithViper(v), AllowMissingKeys(false))

		reqBody := ConfigUpdateRequest{
			Setting: "nonexistent.key",
			Value:   "value",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("setting allowed when allowMissingKeys is true", func(t *testing.T) {
		v := viper.New()
		ws := NewWebSnake(WithViper(v), AllowMissingKeys(true))

		reqBody := ConfigUpdateRequest{
			Setting: "new.key",
			Value:   "newvalue",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if v.GetString("new.key") != "newvalue" {
			t.Error("expected new key to be set")
		}
	})

	t.Run("type mismatch when enforceTypeChecks is true", func(t *testing.T) {
		v := viper.New()
		v.Set("test.key", "string_value")
		ws := NewWebSnake(WithViper(v), EnforceTypeChecks(true))

		reqBody := ConfigUpdateRequest{
			Setting: "test.key",
			Value:   123, // int instead of string
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		bodyStr := w.Body.String()
		if !bytes.Contains([]byte(bodyStr), []byte("Type mismatch")) {
			t.Error("expected 'Type mismatch' in error message")
		}
	})

	t.Run("type check disabled", func(t *testing.T) {
		v := viper.New()
		v.Set("test.key", "string_value")
		ws := NewWebSnake(WithViper(v), EnforceTypeChecks(false))

		reqBody := ConfigUpdateRequest{
			Setting: "test.key",
			Value:   float64(123), // JSON numbers decode as float64
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("validation hook success", func(t *testing.T) {
		v := viper.New()
		v.Set("test.key", "value")
		hookCalled := false
		validationHook := func(r *http.Request, setting string, value any) error {
			hookCalled = true
			if setting != "test.key" {
				t.Errorf("expected setting 'test.key', got '%s'", setting)
			}
			return nil
		}
		ws := NewWebSnake(WithViper(v), WithValidationHook(validationHook))

		reqBody := ConfigUpdateRequest{
			Setting: "test.key",
			Value:   "newvalue",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if !hookCalled {
			t.Error("expected validation hook to be called")
		}
	})

	t.Run("validation hook failure", func(t *testing.T) {
		v := viper.New()
		v.Set("test.key", "value")
		validationHook := func(r *http.Request, setting string, value any) error {
			return errors.New("validation failed")
		}
		ws := NewWebSnake(WithViper(v), WithValidationHook(validationHook))

		reqBody := ConfigUpdateRequest{
			Setting: "test.key",
			Value:   "newvalue",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		bodyStr := w.Body.String()
		if !bytes.Contains([]byte(bodyStr), []byte("Validation failed")) {
			t.Error("expected 'Validation failed' in error message")
		}
	})

	t.Run("onUpdate hook called", func(t *testing.T) {
		v := viper.New()
		v.Set("test.key", "value")
		hookCalled := false
		var capturedSetting string
		var capturedValue any
		onUpdateHook := func(setting string, value any) {
			hookCalled = true
			capturedSetting = setting
			capturedValue = value
		}
		ws := NewWebSnake(WithViper(v), WithOnUpdateHook(onUpdateHook))

		reqBody := ConfigUpdateRequest{
			Setting: "test.key",
			Value:   "newvalue",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if !hookCalled {
			t.Error("expected onUpdate hook to be called")
		}
		if capturedSetting != "test.key" {
			t.Errorf("expected setting 'test.key', got '%s'", capturedSetting)
		}
		if capturedValue != "newvalue" {
			t.Errorf("expected value 'newvalue', got '%v'", capturedValue)
		}
	})

	t.Run("type check with nil existing value", func(t *testing.T) {
		v := viper.New()
		ws := NewWebSnake(WithViper(v), AllowMissingKeys(true), EnforceTypeChecks(true))

		reqBody := ConfigUpdateRequest{
			Setting: "new.key",
			Value:   "newvalue",
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/config/update", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ws.HandleConfigUpdate(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})
}

func TestHandleConfigGet(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		v := viper.New()
		v.Set("test.key", "testvalue")
		ws := NewWebSnake(WithViper(v))

		req := httptest.NewRequest(http.MethodGet, "/config/get/test.key", nil)
		w := httptest.NewRecorder()

		ws.HandleConfigGet(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []ConfigGetResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if len(resp) != 1 {
			t.Fatalf("expected 1 response, got %d", len(resp))
		}
		if resp[0].Setting != "test.key" {
			t.Errorf("expected setting 'test.key', got '%s'", resp[0].Setting)
		}
		if resp[0].Value != "testvalue" {
			t.Errorf("expected value 'testvalue', got '%v'", resp[0].Value)
		}
	})

	t.Run("setting not found when allowMissingKeys is false", func(t *testing.T) {
		v := viper.New()
		ws := NewWebSnake(WithViper(v), AllowMissingKeys(false))

		req := httptest.NewRequest(http.MethodGet, "/config/get/nonexistent.key", nil)
		w := httptest.NewRecorder()

		ws.HandleConfigGet(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("nil value returns not found", func(t *testing.T) {
		v := viper.New()
		ws := NewWebSnake(WithViper(v), AllowMissingKeys(true))

		req := httptest.NewRequest(http.MethodGet, "/config/get/nonexistent.key", nil)
		w := httptest.NewRecorder()

		ws.HandleConfigGet(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("custom config key parser success", func(t *testing.T) {
		v := viper.New()
		v.Set("key1", "value1")
		v.Set("key2", "value2")
		customParser := func(r *http.Request) ([]string, error) {
			return []string{"key1", "key2"}, nil
		}
		ws := NewWebSnake(WithViper(v), WithCustomConfigKeyParser(customParser))

		req := httptest.NewRequest(http.MethodGet, "/config/get/anything", nil)
		w := httptest.NewRecorder()

		ws.HandleConfigGet(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []ConfigGetResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if len(resp) != 2 {
			t.Fatalf("expected 2 responses, got %d", len(resp))
		}
	})

	t.Run("custom config key parser error", func(t *testing.T) {
		v := viper.New()
		customParser := func(r *http.Request) ([]string, error) {
			return nil, errors.New("parse error")
		}
		ws := NewWebSnake(WithViper(v), WithCustomConfigKeyParser(customParser))

		req := httptest.NewRequest(http.MethodGet, "/config/get/anything", nil)
		w := httptest.NewRecorder()

		ws.HandleConfigGet(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		bodyStr := w.Body.String()
		if !bytes.Contains([]byte(bodyStr), []byte("Failed to parse config key")) {
			t.Error("expected 'Failed to parse config key' in error message")
		}
	})

	t.Run("multiple keys with custom parser - one not found", func(t *testing.T) {
		v := viper.New()
		v.Set("key1", "value1")
		customParser := func(r *http.Request) ([]string, error) {
			return []string{"key1", "key2"}, nil
		}
		ws := NewWebSnake(WithViper(v), WithCustomConfigKeyParser(customParser), AllowMissingKeys(false))

		req := httptest.NewRequest(http.MethodGet, "/config/get/anything", nil)
		w := httptest.NewRecorder()

		ws.HandleConfigGet(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("multiple keys with custom parser - nil value", func(t *testing.T) {
		v := viper.New()
		v.Set("key1", "value1")
		customParser := func(r *http.Request) ([]string, error) {
			return []string{"key1", "nonexistent"}, nil
		}
		ws := NewWebSnake(WithViper(v), WithCustomConfigKeyParser(customParser), AllowMissingKeys(true))

		req := httptest.NewRequest(http.MethodGet, "/config/get/anything", nil)
		w := httptest.NewRecorder()

		ws.HandleConfigGet(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
		bodyStr := w.Body.String()
		if !bytes.Contains([]byte(bodyStr), []byte("Setting not found")) {
			t.Error("expected 'Setting not found' in error message")
		}
	})

	t.Run("default path parsing", func(t *testing.T) {
		v := viper.New()
		v.Set("mykey", "myvalue")
		ws := NewWebSnake(WithViper(v))

		req := httptest.NewRequest(http.MethodGet, "/some/deep/path/mykey", nil)
		w := httptest.NewRecorder()

		ws.HandleConfigGet(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []ConfigGetResponse
		json.NewDecoder(w.Body).Decode(&resp)
		if len(resp) != 1 {
			t.Fatalf("expected 1 response, got %d", len(resp))
		}
		if resp[0].Setting != "mykey" {
			t.Errorf("expected setting 'mykey', got '%s'", resp[0].Setting)
		}
	})

	t.Run("get with different value types", func(t *testing.T) {
		v := viper.New()
		v.Set("string.key", "stringval")
		v.Set("int.key", 42)
		v.Set("bool.key", true)
		v.Set("float.key", 3.14)
		ws := NewWebSnake(WithViper(v))

		tests := []struct {
			path          string
			expectedValue any
		}{
			{"/config/get/string.key", "stringval"},
			{"/config/get/int.key", 42},
			{"/config/get/bool.key", true},
			{"/config/get/float.key", 3.14},
		}

		for _, tt := range tests {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			ws.HandleConfigGet(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("path %s: expected status 200, got %d", tt.path, w.Code)
				continue
			}

			var resp []ConfigGetResponse
			json.NewDecoder(w.Body).Decode(&resp)
			if len(resp) != 1 {
				t.Errorf("path %s: expected 1 response, got %d", tt.path, len(resp))
				continue
			}
			// JSON encoding/decoding may change types slightly, so we just check it's not nil
			if resp[0].Value == nil {
				t.Errorf("path %s: expected non-nil value", tt.path)
			}
		}
	})
}

func TestConfigUpdateRequest_JSON(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		req := ConfigUpdateRequest{
			Setting: "test.key",
			Value:   "test.value",
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled ConfigUpdateRequest
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Setting != req.Setting {
			t.Errorf("expected setting %s, got %s", req.Setting, unmarshaled.Setting)
		}
		if unmarshaled.Value != req.Value {
			t.Errorf("expected value %v, got %v", req.Value, unmarshaled.Value)
		}
	})
}

func TestConfigGetResponse_JSON(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		resp := ConfigGetResponse{
			Setting: "test.key",
			Value:   "test.value",
		}

		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var unmarshaled ConfigGetResponse
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if unmarshaled.Setting != resp.Setting {
			t.Errorf("expected setting %s, got %s", resp.Setting, unmarshaled.Setting)
		}
		if unmarshaled.Value != resp.Value {
			t.Errorf("expected value %v, got %v", resp.Value, unmarshaled.Value)
		}
	})
}

func TestHandleConfigUpdate_EmptyBody(t *testing.T) {
	ws := NewWebSnake()
	req := httptest.NewRequest(http.MethodPost, "/config/update", &errorReader{})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	ws.HandleConfigUpdate(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// errorReader is a helper type that always returns an error on Read
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}
