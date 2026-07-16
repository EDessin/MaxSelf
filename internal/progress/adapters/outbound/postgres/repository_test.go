package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/progress/domain"
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

func TestRepositoryGetProfileNotFound(t *testing.T) {
	repo := NewRepository(testDB(t))
	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}

	_, err := repo.GetProfile(context.Background(), "missing-user")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected record not found, got %v", err)
	}
}

func TestRepositorySaveProfileUpsertsProgressAndStats(t *testing.T) {
	repo := NewRepository(testDB(t))
	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}
	last := time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)
	profile := domain.Profile{
		UserID:           "user-1",
		Level:            2,
		TotalXP:          130,
		CurrentLevelXP:   30,
		NextLevelXP:      200,
		StreakDays:       4,
		LastActivityDate: &last,
		Stats: map[domain.Stat]int{
			domain.StatStrength:    40,
			domain.StatConsistency: 5,
		},
	}

	if err := repo.SaveProfile(context.Background(), profile); err != nil {
		t.Fatalf("SaveProfile returned error: %v", err)
	}

	profile.Stats[domain.StatStrength] = 80
	profile.TotalXP = 170
	if err := repo.SaveProfile(context.Background(), profile); err != nil {
		t.Fatalf("second SaveProfile returned error: %v", err)
	}

	got, err := repo.GetProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetProfile returned error: %v", err)
	}
	if got.TotalXP != 170 || got.Stats[domain.StatStrength] != 80 || got.Stats[domain.StatConsistency] != 5 {
		t.Fatalf("unexpected saved profile: %+v", got)
	}
	if got.Stats[domain.StatFuel] != 0 || got.Stats[domain.StatMindset] != 0 || got.Stats[domain.StatRecovery] != 0 {
		t.Fatalf("expected missing stats to default to zero: %+v", got.Stats)
	}
	if got.LastActivityDate == nil || !got.LastActivityDate.Equal(last) {
		t.Fatalf("last activity date not preserved: %v", got.LastActivityDate)
	}

	sqlDB, err := repo.db.DB()
	if err != nil {
		t.Fatalf("unwrap db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}
	if _, err := repo.GetProfile(context.Background(), "user-1"); err == nil {
		t.Fatal("expected get error after closing database")
	}
	if err := repo.SaveProfile(context.Background(), profile); err == nil {
		t.Fatal("expected save error after closing database")
	}
}
