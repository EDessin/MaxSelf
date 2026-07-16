package rest

import (
	"net/http"

	"github.com/EDessin/MaxSelf/internal/facade/application"
	"github.com/EDessin/MaxSelf/internal/platform/config"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
)

type Handler struct {
	service application.Service
	config  config.Config
}

func NewHandler(service application.Service, cfg config.Config) Handler {
	return Handler{service: service, config: cfg}
}

func (h Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "api"})
	})
	mux.HandleFunc("POST /api/auth/register", h.register)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("GET /api/auth/google/login", h.googleLogin)
	mux.HandleFunc("GET /api/me", h.me)
	mux.HandleFunc("GET /api/dashboard", h.dashboard)
	mux.HandleFunc("GET /api/activity-types", h.activityTypes)
	mux.HandleFunc("POST /api/activities", h.createActivity)
	return mux
}

func (h Handler) register(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	result, err := h.service.Register(r.Context(), req)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, result)
}

func (h Handler) login(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	result, err := h.service.Login(r.Context(), req)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

func (h Handler) googleLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, h.config.IdentityServiceURL+"/auth/google/login", http.StatusFound)
}

func (h Handler) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.service.Me(r.Context(), httpx.BearerToken(r))
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	dashboard, err := h.service.Dashboard(r.Context(), httpx.BearerToken(r))
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "could not load dashboard")
		return
	}
	httpx.JSON(w, http.StatusOK, dashboard)
}

func (h Handler) activityTypes(w http.ResponseWriter, r *http.Request) {
	rules, err := h.service.ActivityRules(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load activity types")
		return
	}
	httpx.JSON(w, http.StatusOK, rules)
}

func (h Handler) createActivity(w http.ResponseWriter, r *http.Request) {
	var req map[string]any
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	dashboard, err := h.service.CreateActivity(r.Context(), httpx.BearerToken(r), req)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, dashboard)
}
