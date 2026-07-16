package application

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestClientGetAndPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			http.Error(w, "failed", http.StatusBadGateway)
			return
		}
		if r.Header.Get("X-Test") != "yes" {
			t.Fatalf("missing test header")
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			if r.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("missing JSON content type")
			}
			_, _ = w.Write([]byte(`{"ok":"posted"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":"got"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, time.Second)
	var result map[string]string
	if err := client.Get(t.Context(), "/value", map[string]string{"X-Test": "yes"}, &result); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result["ok"] != "got" {
		t.Fatalf("unexpected get result: %+v", result)
	}

	if err := client.Post(t.Context(), "/value", map[string]string{"X-Test": "yes"}, map[string]string{"name": "Max"}, &result); err != nil {
		t.Fatalf("Post returned error: %v", err)
	}
	if result["ok"] != "posted" {
		t.Fatalf("unexpected post result: %+v", result)
	}

	if err := client.Get(t.Context(), "/fail", map[string]string{"X-Test": "yes"}, &result); err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected status error, got %v", err)
	}

	if err := client.Post(t.Context(), "/value", nil, func() {}, &result); err == nil {
		t.Fatal("expected marshal error")
	}
}
