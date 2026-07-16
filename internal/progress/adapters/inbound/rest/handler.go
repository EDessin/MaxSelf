package rest

import (
	"net/http"
	"strings"
	"time"

	"github.com/EDessin/MaxSelf/internal/platform/httpx"
	"github.com/EDessin/MaxSelf/internal/progress/application"
	"github.com/EDessin/MaxSelf/internal/progress/domain"
)

type Handler struct {
	service application.Service
}

func NewHandler(service application.Service) Handler {
	return Handler{service: service}
}

func (h Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "progress"})
	})
	mux.HandleFunc("POST /progress/award", h.award)
	mux.HandleFunc("GET /progress/", h.get)
	return mux
}

type awardRequest struct {
	UserID     string      `json:"userId"`
	ActivityID string      `json:"activityId"`
	XP         int         `json:"xp"`
	Stat       domain.Stat `json:"stat"`
	OccurredAt time.Time   `json:"occurredAt"`
}

func (h Handler) award(w http.ResponseWriter, r *http.Request) {
	var req awardRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	profile, err := h.service.Award(r.Context(), domain.Award(req))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not award xp")
		return
	}
	httpx.JSON(w, http.StatusOK, profile)
}

func (h Handler) get(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimPrefix(r.URL.Path, "/progress/")
	if userID == "" {
		httpx.Error(w, http.StatusBadRequest, "missing user id")
		return
	}
	profile, err := h.service.Get(r.Context(), userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load progress")
		return
	}
	httpx.JSON(w, http.StatusOK, profile)
}
