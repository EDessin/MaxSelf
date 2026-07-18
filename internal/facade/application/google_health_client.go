package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var googleHealthScopes = []string{
	"https://www.googleapis.com/auth/googlehealth.activity_and_fitness.readonly",
	"https://www.googleapis.com/auth/googlehealth.health_metrics_and_measurements.readonly",
	"https://www.googleapis.com/auth/googlehealth.nutrition.readonly",
	"https://www.googleapis.com/auth/googlehealth.sleep.readonly",
}

const maxGoogleHealthErrorBodyBytes = 4096
const maxGoogleHealthRequestAttempts = 2

type GoogleHealthHTTPClient struct {
	oauth  *oauth2.Config
	apiURL string
	http   *http.Client
}

func NewGoogleHealthHTTPClient(clientID string, clientSecret string, redirectURL string, apiURL string, timeout time.Duration) *GoogleHealthHTTPClient {
	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return nil
	}
	if apiURL == "" {
		apiURL = "https://health.googleapis.com"
	}
	return &GoogleHealthHTTPClient{
		oauth: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       googleHealthScopes,
			Endpoint:     google.Endpoint,
		},
		apiURL: strings.TrimRight(apiURL, "/"),
		http:   &http.Client{Timeout: timeout},
	}
}

func (c *GoogleHealthHTTPClient) AuthCodeURL(state string) string {
	return c.oauth.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
		oauth2.SetAuthURLParam("include_granted_scopes", "true"),
	)
}

func (c *GoogleHealthHTTPClient) ExchangeCode(ctx context.Context, code string) (HealthConnection, error) {
	token, err := c.oauth.Exchange(ctx, code)
	if err != nil {
		log.Printf("google health oauth code exchange failed: %v", err)
		return HealthConnection{}, err
	}
	return connectionFromOAuthToken(token), nil
}

func (c *GoogleHealthHTTPClient) Reconcile(ctx context.Context, connection HealthConnection, dataType string, filter string) (HealthConnection, []HealthDataPoint, error) {
	token, err := c.oauth.TokenSource(ctx, oauthTokenFromConnection(connection)).Token()
	if err != nil {
		log.Printf("google health token refresh failed user_id=%s data_type=%s err=%v", connection.UserID, dataType, err)
		return connection, nil, fmt.Errorf("google health token refresh failed: %w", err)
	}
	connection = mergeOAuthToken(connection, token)

	var all []HealthDataPoint
	pageToken := ""
	for {
		endpoint, err := url.Parse(fmt.Sprintf("%s/v4/users/me/dataTypes/%s/dataPoints:reconcile", c.apiURL, dataType))
		if err != nil {
			return connection, nil, err
		}
		query := endpoint.Query()
		query.Set("pageSize", "25")
		if filter != "" {
			query.Set("filter", filter)
		}
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		endpoint.RawQuery = query.Encode()

		resp, err := c.doReconcileRequest(ctx, endpoint.String(), token.AccessToken, connection.UserID, dataType, filter)
		if err != nil {
			return connection, nil, fmt.Errorf("google health %s reconcile request failed: %w", dataType, err)
		}
		if resp.StatusCode >= 400 {
			body := readGoogleHealthErrorBody(resp.Body)
			_ = resp.Body.Close()
			log.Printf("google health reconcile returned error user_id=%s data_type=%s filter=%q status=%s body=%q", connection.UserID, dataType, filter, resp.Status, body)
			if body != "" {
				return connection, nil, fmt.Errorf("google health %s reconcile failed: %s: %s", dataType, resp.Status, body)
			}
			return connection, nil, fmt.Errorf("google health %s reconcile failed: %s", dataType, resp.Status)
		}
		var payload struct {
			DataPoints    []HealthDataPoint `json:"dataPoints"`
			NextPageToken string            `json:"nextPageToken"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			_ = resp.Body.Close()
			log.Printf("google health reconcile decode failed user_id=%s data_type=%s status=%s err=%v", connection.UserID, dataType, resp.Status, err)
			return connection, nil, fmt.Errorf("google health %s reconcile decode failed: %w", dataType, err)
		}
		_ = resp.Body.Close()
		all = append(all, payload.DataPoints...)
		if payload.NextPageToken == "" {
			return connection, all, nil
		}
		pageToken = payload.NextPageToken
	}
}

func (c *GoogleHealthHTTPClient) doReconcileRequest(ctx context.Context, endpoint string, accessToken string, userID string, dataType string, filter string) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= maxGoogleHealthRequestAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := c.http.Do(req)
		if err == nil {
			if attempt > 1 {
				log.Printf("google health reconcile retry succeeded user_id=%s data_type=%s attempt=%d", userID, dataType, attempt)
			}
			return resp, nil
		}
		lastErr = err
		log.Printf("google health reconcile request failed user_id=%s data_type=%s filter=%q attempt=%d err=%v", userID, dataType, filter, attempt, err)
		if !isRetryableGoogleHealthRequestError(err) || attempt == maxGoogleHealthRequestAttempts {
			break
		}
		log.Printf("google health reconcile request retrying user_id=%s data_type=%s next_attempt=%d", userID, dataType, attempt+1)
	}
	return nil, lastErr
}

func isRetryableGoogleHealthRequestError(err error) bool {
	return os.IsTimeout(err) || errors.Is(err, context.DeadlineExceeded)
}

func readGoogleHealthErrorBody(body io.Reader) string {
	payload, err := io.ReadAll(io.LimitReader(body, maxGoogleHealthErrorBodyBytes))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(payload))
}

func oauthTokenFromConnection(connection HealthConnection) *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  connection.AccessToken,
		RefreshToken: connection.RefreshToken,
		TokenType:    connection.TokenType,
		Expiry:       connection.Expiry,
	}
}

func connectionFromOAuthToken(token *oauth2.Token) HealthConnection {
	connection := HealthConnection{}
	return mergeOAuthToken(connection, token)
}

func mergeOAuthToken(connection HealthConnection, token *oauth2.Token) HealthConnection {
	connection.AccessToken = token.AccessToken
	if token.RefreshToken != "" {
		connection.RefreshToken = token.RefreshToken
	}
	connection.TokenType = token.TokenType
	connection.Expiry = token.Expiry
	if scopes, ok := token.Extra("scope").(string); ok {
		connection.Scope = scopes
	}
	return connection
}
