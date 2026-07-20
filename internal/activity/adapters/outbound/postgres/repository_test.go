package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/activity/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

func TestRepositoryCreateAndListByUser(t *testing.T) {
	repo := NewRepository(testDB(t))
	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}

	older := domain.Activity{
		ID:         "activity-1",
		UserID:     "user-1",
		Type:       domain.TypeExercise,
		Title:      "Resistance Training",
		Notes:      "run",
		XP:         40,
		Stat:       domain.StatStrength,
		OccurredAt: time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC),
	}
	newer := older
	newer.ID = "activity-2"
	newer.OccurredAt = older.OccurredAt.Add(24 * time.Hour)
	otherUser := older
	otherUser.ID = "activity-3"
	otherUser.UserID = "user-2"

	for _, activity := range []domain.Activity{older, newer, otherUser} {
		created, err := repo.Create(context.Background(), activity)
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		if created.ID != activity.ID || created.CreatedAt.IsZero() {
			t.Fatalf("unexpected created activity: %+v", created)
		}
	}

	activities, err := repo.ListByUser(context.Background(), "user-1", 10)
	if err != nil {
		t.Fatalf("ListByUser returned error: %v", err)
	}
	if len(activities) != 2 || activities[0].ID != "activity-2" || activities[1].ID != "activity-1" {
		t.Fatalf("unexpected activities order/list: %+v", activities)
	}

	limited, err := repo.ListByUser(context.Background(), "user-1", 1)
	if err != nil {
		t.Fatalf("limited ListByUser returned error: %v", err)
	}
	if len(limited) != 1 || limited[0].ID != "activity-2" {
		t.Fatalf("unexpected limited activities: %+v", limited)
	}

	sqlDB, err := repo.db.DB()
	if err != nil {
		t.Fatalf("unwrap db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	if _, err := repo.Create(context.Background(), older); err == nil {
		t.Fatal("expected create error after closing database")
	}
	if _, err := repo.ListByUser(context.Background(), "user-1", 10); err == nil {
		t.Fatal("expected list error after closing database")
	}
}
