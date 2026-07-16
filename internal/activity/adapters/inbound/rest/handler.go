package rest

import (
	"net/http"
	"strconv"
	"time"

	"github.com/EDessin/MaxSelf/internal/activity/application"
	"github.com/EDessin/MaxSelf/internal/activity/domain"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
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
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "activity"})
	})
	mux.HandleFunc("GET /activity-types", h.rules)
	mux.HandleFunc("POST /activities", h.create)
	mux.HandleFunc("GET /activities", h.list)
	return mux
}

type createRequest struct {
	Type       domain.ActivityType `json:"type"`
	Notes      string              `json:"notes"`
	OccurredAt *time.Time          `json:"occurredAt"`
}

func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		httpx.Error(w, http.StatusUnauthorized, "missing user")
		return
	}
	var req createRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	var occurredAt time.Time
	if req.OccurredAt != nil {
		occurredAt = *req.OccurredAt
	}
	activity, err := h.service.Create(r.Context(), userID, req.Type, req.Notes, occurredAt)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, activity)
}

func (h Handler) list(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		httpx.Error(w, http.StatusUnauthorized, "missing user")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	activities, err := h.service.List(r.Context(), userID, limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "could not load activities")
		return
	}
	httpx.JSON(w, http.StatusOK, activities)
}

func (h Handler) rules(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, h.service.Rules())
}
