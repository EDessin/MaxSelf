package rest

import (
	"errors"
	"fmt"
	"log"
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
	mux.HandleFunc("POST /api/integrations/google-health/connect", h.googleHealthConnect)
	mux.HandleFunc("GET /api/integrations/google-health/callback", h.googleHealthCallback)
	mux.HandleFunc("POST /api/integrations/google-health/sync", h.googleHealthSync)
	mux.HandleFunc("POST /api/biometrics/waist-to-height", h.waistToHeight)
	mux.HandleFunc("POST /api/quest-claims/{claimID}/claim", h.claimQuest)
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
	httpx.Error(w, http.StatusGone, "manual XP claims are disabled; sync health data and claim a quest instead")
}

func (h Handler) googleHealthConnect(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.StartGoogleHealthConnect(r.Context(), httpx.BearerToken(r))
	if err != nil {
		log.Printf("google health connect route failed remote_addr=%s err=%v", r.RemoteAddr, err)
		h.integrationError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

func (h Handler) googleHealthCallback(w http.ResponseWriter, r *http.Request) {
	if err := h.service.CompleteGoogleHealthConnect(r.Context(), r.URL.Query().Get("state"), r.URL.Query().Get("code")); err != nil {
		log.Printf("google health callback route failed remote_addr=%s state=%s err=%v", r.RemoteAddr, r.URL.Query().Get("state"), err)
		http.Redirect(w, r, fmt.Sprintf("%s/?googleHealth=error", h.config.FrontendURL), http.StatusFound)
		return
	}
	log.Printf("google health callback route completed remote_addr=%s state=%s", r.RemoteAddr, r.URL.Query().Get("state"))
	http.Redirect(w, r, fmt.Sprintf("%s/?googleHealth=connected", h.config.FrontendURL), http.StatusFound)
}

func (h Handler) googleHealthSync(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.SyncGoogleHealth(r.Context(), httpx.BearerToken(r))
	if err != nil {
		log.Printf("google health sync route failed remote_addr=%s err=%v", r.RemoteAddr, err)
		h.integrationError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

func (h Handler) waistToHeight(w http.ResponseWriter, r *http.Request) {
	var req application.WaistToHeightRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	result, err := h.service.CreateWaistToHeightClaim(r.Context(), httpx.BearerToken(r), req)
	if err != nil {
		log.Printf("waist-to-height route failed remote_addr=%s err=%v", r.RemoteAddr, err)
		h.integrationError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, result)
}

func (h Handler) claimQuest(w http.ResponseWriter, r *http.Request) {
	dashboard, err := h.service.ClaimQuest(r.Context(), httpx.BearerToken(r), r.PathValue("claimID"))
	if err != nil {
		log.Printf("quest claim route failed remote_addr=%s claim_id=%s err=%v", r.RemoteAddr, r.PathValue("claimID"), err)
		h.integrationError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, dashboard)
}

func (h Handler) integrationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrGoogleHealthNotConfigured):
		httpx.Error(w, http.StatusNotImplemented, err.Error())
	case errors.Is(err, application.ErrGoogleHealthNotConnected):
		httpx.Error(w, http.StatusConflict, err.Error())
	case errors.Is(err, application.ErrQuestClaimNotFound):
		httpx.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, application.ErrQuestClaimAlreadyClaimed):
		httpx.Error(w, http.StatusConflict, err.Error())
	default:
		httpx.Error(w, http.StatusBadRequest, err.Error())
	}
}
