package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONAndError(t *testing.T) {
	recorder := httptest.NewRecorder()
	JSON(recorder, http.StatusCreated, map[string]string{"ok": "yes"})

	if recorder.Code != http.StatusCreated || recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("unexpected JSON response: code=%d headers=%v", recorder.Code, recorder.Header())
	}
	var payload map[string]string
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode JSON response: %v", err)
	}
	if payload["ok"] != "yes" {
		t.Fatalf("unexpected payload: %+v", payload)
	}

	recorder = httptest.NewRecorder()
	Error(recorder, http.StatusTeapot, "short and stout")
	if recorder.Code != http.StatusTeapot || !strings.Contains(recorder.Body.String(), "short and stout") {
		t.Fatalf("unexpected error response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	var target struct {
		Name string `json:"name"`
	}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Max"}`))
	if err := Decode(request, &target); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if target.Name != "Max" {
		t.Fatalf("unexpected decoded target: %+v", target)
	}

	request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Max","extra":true}`))
	if err := Decode(request, &target); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestBearerToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	if token := BearerToken(request); token != "" {
		t.Fatalf("expected empty token, got %q", token)
	}

	request.Header.Set("Authorization", "Basic nope")
	if token := BearerToken(request); token != "" {
		t.Fatalf("expected empty token for non-bearer header, got %q", token)
	}

	request.Header.Set("Authorization", "Bearer  token-1 ")
	if token := BearerToken(request); token != "token-1" {
		t.Fatalf("unexpected bearer token: %q", token)
	}
}

func TestWithCORS(t *testing.T) {
	called := false
	handler := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	}), "https://app.example.com")

	request := httptest.NewRequest(http.MethodOptions, "/", nil)
	request.Header.Set("Origin", "https://app.example.com")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent || recorder.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("unexpected preflight response: %d %v", recorder.Code, recorder.Header())
	}
	if called {
		t.Fatal("preflight should not call next handler")
	}

	request = httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Origin", "http://localhost:4200")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted || recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:4200" {
		t.Fatalf("unexpected CORS passthrough: %d %v", recorder.Code, recorder.Header())
	}
}
