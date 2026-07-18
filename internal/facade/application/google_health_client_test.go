package application

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestGoogleHealthHTTPClientOAuthAndReconcile(t *testing.T) {
	tokenCalls := 0
	apiCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			tokenCalls++
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			switch r.FormValue("grant_type") {
			case "authorization_code":
				if r.FormValue("code") != "code-1" {
					t.Fatalf("unexpected auth code token request: %s", r.Form.Encode())
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"access-1","refresh_token":"refresh-1","token_type":"Bearer","expires_in":3600,"scope":"scope-a scope-b"}`))
			case "refresh_token":
				if r.FormValue("refresh_token") != "refresh-1" {
					t.Fatalf("unexpected refresh token request: %s", r.Form.Encode())
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"access-2","token_type":"Bearer","expires_in":3600}`))
			default:
				t.Fatalf("unexpected token grant: %s", r.FormValue("grant_type"))
			}
		case "/v4/users/me/dataTypes/exercise/dataPoints:reconcile":
			apiCalls++
			if r.Header.Get("Authorization") != "Bearer access-2" {
				t.Fatalf("unexpected authorization header: %s", r.Header.Get("Authorization"))
			}
			if r.URL.Query().Get("pageSize") != "25" || r.URL.Query().Get("filter") != `exercise.interval.civil_start_time >= "2026-07-17T00:00:00"` {
				t.Fatalf("unexpected reconcile query: %s", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("pageToken") == "" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"dataPoints": []map[string]any{{
						"dataPointName": "point-1",
						"exercise": map[string]any{
							"exerciseType": "RUNNING",
						},
					}},
					"nextPageToken": "next",
				})
				return
			}
			if r.URL.Query().Get("pageToken") != "next" {
				t.Fatalf("unexpected page token: %s", r.URL.Query().Get("pageToken"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"dataPoints": []map[string]any{{
					"dataPointName": "point-2",
					"exercise": map[string]any{
						"exerciseType": "CYCLING",
					},
				}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewGoogleHealthHTTPClient("client-id", "client-secret", "http://localhost/callback", server.URL+"/", time.Second)
	if client == nil {
		t.Fatal("expected configured Google Health client")
	}
	client.oauth.Endpoint = oauth2.Endpoint{AuthURL: server.URL + "/auth", TokenURL: server.URL + "/token"}
	client.http = server.Client()
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, server.Client())

	authURL, err := url.Parse(client.AuthCodeURL("state-1"))
	if err != nil {
		t.Fatalf("parse auth URL: %v", err)
	}
	if authURL.Query().Get("state") != "state-1" ||
		authURL.Query().Get("access_type") != "offline" ||
		authURL.Query().Get("prompt") != "consent" ||
		authURL.Query().Get("include_granted_scopes") != "true" ||
		!strings.Contains(authURL.Query().Get("scope"), "googlehealth.sleep.readonly") {
		t.Fatalf("unexpected auth URL: %s", authURL.String())
	}

	connection, err := client.ExchangeCode(ctx, "code-1")
	if err != nil {
		t.Fatalf("ExchangeCode returned error: %v", err)
	}
	if connection.AccessToken != "access-1" || connection.RefreshToken != "refresh-1" || connection.Scope != "scope-a scope-b" {
		t.Fatalf("unexpected exchanged connection: %+v", connection)
	}

	connection.Expiry = time.Now().Add(-time.Minute)
	connection, points, err := client.Reconcile(ctx, connection, "exercise", `exercise.interval.civil_start_time >= "2026-07-17T00:00:00"`)
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if connection.AccessToken != "access-2" || connection.RefreshToken != "refresh-1" {
		t.Fatalf("unexpected refreshed connection: %+v", connection)
	}
	if len(points) != 2 || points[0].DataPointName != "point-1" || points[1].Exercise.ExerciseType != "CYCLING" {
		t.Fatalf("unexpected reconciled points: %+v", points)
	}
	if tokenCalls != 2 || apiCalls != 2 {
		t.Fatalf("unexpected call counts token=%d api=%d", tokenCalls, apiCalls)
	}
}

func TestGoogleHealthHTTPClientRetriesRequestTimeout(t *testing.T) {
	calls := 0
	client := NewGoogleHealthHTTPClient("client", "secret", "callback", "https://health.example", time.Second)
	client.http = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return nil, context.DeadlineExceeded
			}
			if req.Header.Get("Authorization") != "Bearer access" {
				t.Fatalf("unexpected authorization header: %s", req.Header.Get("Authorization"))
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"dataPoints":[{"dataPointName":"steps-1","steps":{"count":"40"}}]}`)),
			}, nil
		}),
	}

	_, points, err := client.Reconcile(context.Background(), HealthConnection{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}, "steps", "")
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if calls != 2 || len(points) != 1 || points[0].Steps.Count != "40" {
		t.Fatalf("unexpected retry result calls=%d points=%+v", calls, points)
	}
}

func TestGoogleHealthHTTPClientConfigurationAndErrors(t *testing.T) {
	if NewGoogleHealthHTTPClient("", "secret", "callback", "", time.Second) != nil {
		t.Fatal("expected missing client id to disable Google Health client")
	}
	client := NewGoogleHealthHTTPClient("client", "secret", "callback", "", time.Second)
	if client == nil || client.apiURL != "https://health.googleapis.com" {
		t.Fatalf("unexpected default client: %+v", client)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer server.Close()
	client.apiURL = server.URL
	client.http = server.Client()
	client.oauth.Endpoint = oauth2.Endpoint{AuthURL: server.URL + "/auth", TokenURL: server.URL + "/token"}

	_, _, err := client.Reconcile(context.Background(), HealthConnection{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}, "sleep", "")
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected google health status error, got %v", err)
	}

	token := oauthTokenFromConnection(HealthConnection{AccessToken: "a", RefreshToken: "r", TokenType: "Bearer"})
	if token.AccessToken != "a" || token.RefreshToken != "r" || token.TokenType != "Bearer" {
		t.Fatalf("unexpected oauth token: %+v", token)
	}
	merged := mergeOAuthToken(HealthConnection{RefreshToken: "old-refresh"}, &oauth2.Token{AccessToken: "new-access", TokenType: "Bearer"})
	if merged.AccessToken != "new-access" || merged.RefreshToken != "old-refresh" {
		t.Fatalf("expected merge to preserve refresh token: %+v", merged)
	}
}

func TestGoogleHealthHTTPClientExchangeAndDecodeErrors(t *testing.T) {
	tokenStatus := http.StatusBadRequest
	apiBody := `not-json`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			http.Error(w, "token failed", tokenStatus)
		case "/v4/users/me/dataTypes/sleep/dataPoints:reconcile":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(apiBody))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewGoogleHealthHTTPClient("client", "secret", "callback", server.URL, time.Second)
	client.oauth.Endpoint = oauth2.Endpoint{AuthURL: server.URL + "/auth", TokenURL: server.URL + "/token"}
	client.http = server.Client()
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, server.Client())

	if _, err := client.ExchangeCode(ctx, "bad-code"); err == nil {
		t.Fatal("expected ExchangeCode token error")
	}
	if _, _, err := client.Reconcile(ctx, HealthConnection{RefreshToken: "refresh", TokenType: "Bearer", Expiry: time.Now().Add(-time.Hour)}, "sleep", ""); err == nil {
		t.Fatal("expected refresh token error")
	}

	tokenStatus = http.StatusOK
	if _, _, err := client.Reconcile(ctx, HealthConnection{AccessToken: "access", RefreshToken: "refresh", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}, "sleep", ""); err == nil {
		t.Fatal("expected reconcile JSON decode error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
