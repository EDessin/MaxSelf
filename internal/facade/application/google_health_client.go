package application

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
		return HealthConnection{}, err
	}
	return connectionFromOAuthToken(token), nil
}

func (c *GoogleHealthHTTPClient) Reconcile(ctx context.Context, connection HealthConnection, dataType string, filter string) (HealthConnection, []HealthDataPoint, error) {
	token, err := c.oauth.TokenSource(ctx, oauthTokenFromConnection(connection)).Token()
	if err != nil {
		return connection, nil, err
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

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return connection, nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+token.AccessToken)

		resp, err := c.http.Do(req)
		if err != nil {
			return connection, nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			return connection, nil, fmt.Errorf("google health returned %s", resp.Status)
		}
		var payload struct {
			DataPoints    []HealthDataPoint `json:"dataPoints"`
			NextPageToken string            `json:"nextPageToken"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			return connection, nil, err
		}
		all = append(all, payload.DataPoints...)
		if payload.NextPageToken == "" {
			return connection, all, nil
		}
		pageToken = payload.NextPageToken
	}
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
