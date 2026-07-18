package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/activity/application"
	"github.com/EDessin/MaxSelf/internal/activity/domain"
)

type fakeActivityRepository struct {
	createErr error
	listErr   error
	created   domain.Activity
	listLimit int
}

func (r *fakeActivityRepository) Create(_ context.Context, activity domain.Activity) (domain.Activity, error) {
	r.created = activity
	if r.createErr != nil {
		return domain.Activity{}, r.createErr
	}
	return activity, nil
}

func (r *fakeActivityRepository) ListByUser(_ context.Context, _ string, limit int) ([]domain.Activity, error) {
	r.listLimit = limit
	if r.listErr != nil {
		return nil, r.listErr
	}
	return []domain.Activity{{
		ID:         "activity-1",
		UserID:     "user-1",
		Type:       domain.TypeExercise,
		Title:      "Strength Session",
		XP:         40,
		Stat:       domain.StatStrength,
		OccurredAt: time.Now(),
	}}, nil
}

func TestRoutesHealthAndRules(t *testing.T) {
	handler := NewHandler(application.NewService(&fakeActivityRepository{})).Routes()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "activity") {
		t.Fatalf("unexpected health response: %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/activity-types", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected rules response: %d %s", recorder.Code, recorder.Body.String())
	}
	var rules []domain.ActivityRule
	if err := json.NewDecoder(recorder.Body).Decode(&rules); err != nil {
		t.Fatalf("decode rules: %v", err)
	}
	if len(rules) != 9 {
		t.Fatalf("expected 9 rules, got %d", len(rules))
	}
	if rules[7].Type != domain.TypeScaleMeasurement || rules[8].Type != domain.TypeWaistToHeightRatio {
		t.Fatalf("unexpected biometrics rules: %+v", rules[7:])
	}
}

func TestCreateActivityRoute(t *testing.T) {
	repo := &fakeActivityRepository{}
	handler := NewHandler(application.NewService(repo)).Routes()
	body := `{"type":"exercise","notes":"ran"}`
	request := httptest.NewRequest(http.MethodPost, "/activities", strings.NewReader(body))
	request.Header.Set("X-User-ID", "user-1")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("unexpected create response: %d %s", recorder.Code, recorder.Body.String())
	}
	if repo.created.UserID != "user-1" || repo.created.Notes != "ran" || repo.created.XP != 40 {
		t.Fatalf("unexpected created activity: %+v", repo.created)
	}
}

func TestCreateActivityRouteValidation(t *testing.T) {
	handler := NewHandler(application.NewService(&fakeActivityRepository{})).Routes()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/activities", strings.NewReader(`{"type":"exercise"}`)))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing user unauthorized, got %d", recorder.Code)
	}

	request := httptest.NewRequest(http.MethodPost, "/activities", strings.NewReader(`{"type":"exercise","extra":true}`))
	request.Header.Set("X-User-ID", "user-1")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid request, got %d", recorder.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/activities", strings.NewReader(`{"type":"unknown"}`))
	request.Header.Set("X-User-ID", "user-1")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected unknown type bad request, got %d", recorder.Code)
	}
}

func TestListActivitiesRoute(t *testing.T) {
	repo := &fakeActivityRepository{}
	handler := NewHandler(application.NewService(repo)).Routes()
	request := httptest.NewRequest(http.MethodGet, "/activities?limit=7", nil)
	request.Header.Set("X-User-ID", "user-1")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK || repo.listLimit != 7 {
		t.Fatalf("unexpected list response: code=%d limit=%d body=%s", recorder.Code, repo.listLimit, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/activities", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing user unauthorized, got %d", recorder.Code)
	}

	repo.listErr = errors.New("store failed")
	request = httptest.NewRequest(http.MethodGet, "/activities", nil)
	request.Header.Set("X-User-ID", "user-1")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected list error 500, got %d", recorder.Code)
	}
}
