package rest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/progress/application"
	"github.com/EDessin/MaxSelf/internal/progress/domain"
)

type fakeProgressRepository struct {
	profile domain.Profile
	getErr  error
	saveErr error
}

func (r *fakeProgressRepository) GetProfile(_ context.Context, userID string) (domain.Profile, error) {
	if r.getErr != nil {
		return domain.Profile{}, r.getErr
	}
	if r.profile.UserID == "" {
		r.profile = domain.Profile{UserID: userID, Level: 1, NextLevelXP: 100, Stats: map[domain.Stat]int{}}
	}
	return r.profile, nil
}

func (r *fakeProgressRepository) SaveProfile(_ context.Context, profile domain.Profile) error {
	r.profile = profile
	return r.saveErr
}

func TestRoutesHealthAwardAndGet(t *testing.T) {
	repo := &fakeProgressRepository{profile: domain.Profile{UserID: "user-1", Level: 1, NextLevelXP: 100, Stats: map[domain.Stat]int{}}}
	handler := NewHandler(application.NewService(repo)).Routes()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "progress") {
		t.Fatalf("unexpected health response: %d %s", recorder.Code, recorder.Body.String())
	}

	body := `{"userId":"user-1","activityId":"activity-1","xp":40,"stat":"strength","occurredAt":"` + time.Now().UTC().Format(time.RFC3339) + `"}`
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/progress/award", strings.NewReader(body)))
	if recorder.Code != http.StatusOK || repo.profile.TotalXP != 40 {
		t.Fatalf("unexpected award response: %d body=%s profile=%+v", recorder.Code, recorder.Body.String(), repo.profile)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/progress/user-1", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"userId":"user-1"`) {
		t.Fatalf("unexpected get response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestRoutesReturnErrors(t *testing.T) {
	handler := NewHandler(application.NewService(&fakeProgressRepository{})).Routes()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/progress/award", strings.NewReader(`{"extra":true}`)))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected bad award request, got %d", recorder.Code)
	}

	repo := &fakeProgressRepository{saveErr: errors.New("save failed")}
	handler = NewHandler(application.NewService(repo)).Routes()
	body := `{"userId":"user-1","activityId":"activity-1","xp":40,"stat":"strength","occurredAt":"` + time.Now().UTC().Format(time.RFC3339) + `"}`
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/progress/award", strings.NewReader(body)))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected award save error, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/progress/", nil))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected missing user id, got %d", recorder.Code)
	}

	repo = &fakeProgressRepository{getErr: errors.New("get failed")}
	handler = NewHandler(application.NewService(repo)).Routes()
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/progress/user-1", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected fallback profile response, got %d", recorder.Code)
	}
}
