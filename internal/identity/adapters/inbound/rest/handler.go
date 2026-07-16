package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/EDessin/MaxSelf/internal/identity/application"
	"github.com/EDessin/MaxSelf/internal/identity/domain"
	"github.com/EDessin/MaxSelf/internal/platform/config"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

type Handler struct {
	service application.Service
	config  config.Config
	oauth   *oauth2.Config
}

func NewHandler(service application.Service, cfg config.Config) Handler {
	return Handler{
		service: service,
		config:  cfg,
		oauth: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		},
	}
}

func (h Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "identity"})
	})
	mux.HandleFunc("POST /auth/register", h.register)
	mux.HandleFunc("POST /auth/login", h.login)
	mux.HandleFunc("GET /auth/google/login", h.googleLogin)
	mux.HandleFunc("GET /auth/google/callback", h.googleCallback)
	mux.HandleFunc("GET /users/me", h.me)
	return mux
}

type authRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

func (h Handler) register(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	result, err := h.service.Register(r.Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, result)
}

func (h Handler) login(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	result, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

func (h Handler) googleLogin(w http.ResponseWriter, r *http.Request) {
	if h.config.GoogleClientID == "" || h.config.GoogleClientSecret == "" {
		httpx.Error(w, http.StatusNotImplemented, "google login is not configured")
		return
	}
	state := "maxself-local"
	http.Redirect(w, r, h.oauth.AuthCodeURL(state, oauth2.AccessTypeOnline), http.StatusFound)
}

func (h Handler) googleCallback(w http.ResponseWriter, r *http.Request) {
	if h.config.GoogleClientID == "" || h.config.GoogleClientSecret == "" {
		httpx.Error(w, http.StatusNotImplemented, "google login is not configured")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, h.config.FrontendURL+"/login?error=google", http.StatusFound)
		return
	}
	token, err := h.oauth.Exchange(r.Context(), code)
	if err != nil {
		http.Redirect(w, r, h.config.FrontendURL+"/login?error=google", http.StatusFound)
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		http.Redirect(w, r, h.config.FrontendURL+"/login?error=google", http.StatusFound)
		return
	}
	payload, err := idtoken.Validate(context.Background(), rawIDToken, h.config.GoogleClientID)
	if err != nil {
		http.Redirect(w, r, h.config.FrontendURL+"/login?error=google", http.StatusFound)
		return
	}

	email, _ := payload.Claims["email"].(string)
	name, _ := payload.Claims["name"].(string)
	picture, _ := payload.Claims["picture"].(string)
	result, err := h.service.LoginWithProvider(r.Context(), domain.ProviderGoogle, payload.Subject, email, name, picture)
	if err != nil {
		http.Redirect(w, r, h.config.FrontendURL+"/login?error=google", http.StatusFound)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("%s/auth/callback?token=%s", h.config.FrontendURL, result.Token), http.StatusFound)
}

func (h Handler) me(w http.ResponseWriter, r *http.Request) {
	claims, err := httpx.ParseToken(h.config.JWTSecret, httpx.BearerToken(r))
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := h.service.Me(r.Context(), claims)
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "user not found")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}
