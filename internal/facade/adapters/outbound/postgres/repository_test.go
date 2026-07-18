package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/EDessin/MaxSelf/internal/facade/application"
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

func TestRepositoryAuthStatesConnectionsAndQuestClaims(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(testDB(t))
	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}

	now := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	if err := repo.SaveHealthAuthState(ctx, application.HealthAuthState{
		State:     "state-1",
		UserID:    "user-1",
		ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("SaveHealthAuthState returned error: %v", err)
	}

	consumed, err := repo.ConsumeHealthAuthState(ctx, "state-1", now)
	if err != nil {
		t.Fatalf("ConsumeHealthAuthState returned error: %v", err)
	}
	if consumed.UserID != "user-1" || consumed.UsedAt == nil || !consumed.UsedAt.Equal(now) {
		t.Fatalf("unexpected consumed auth state: %+v", consumed)
	}
	if _, err := repo.ConsumeHealthAuthState(ctx, "state-1", now); err == nil {
		t.Fatal("expected used auth state to fail")
	}

	if err := repo.SaveHealthAuthState(ctx, application.HealthAuthState{
		State:     "state-2",
		UserID:    "user-1",
		ExpiresAt: now.Add(-time.Minute),
	}); err != nil {
		t.Fatalf("SaveHealthAuthState expired returned error: %v", err)
	}
	if _, err := repo.ConsumeHealthAuthState(ctx, "state-2", now); err == nil {
		t.Fatal("expected expired auth state to fail")
	}

	connection := application.HealthConnection{
		UserID:       "user-1",
		AccessToken:  "access-1",
		RefreshToken: "refresh-1",
		TokenType:    "Bearer",
		Scope:        "scope-a",
		Expiry:       now.Add(time.Hour),
	}
	if err := repo.SaveHealthConnection(ctx, connection); err != nil {
		t.Fatalf("SaveHealthConnection returned error: %v", err)
	}
	syncedAt := now.Add(30 * time.Minute)
	connection.AccessToken = "access-2"
	connection.Scope = "scope-b"
	connection.LastSyncedAt = &syncedAt
	if err := repo.SaveHealthConnection(ctx, connection); err != nil {
		t.Fatalf("second SaveHealthConnection returned error: %v", err)
	}
	gotConnection, err := repo.GetHealthConnection(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetHealthConnection returned error: %v", err)
	}
	if gotConnection.AccessToken != "access-2" || gotConnection.Scope != "scope-b" || gotConnection.LastSyncedAt == nil {
		t.Fatalf("unexpected saved connection: %+v", gotConnection)
	}
	latestSync := now.Add(time.Hour)
	if err := repo.UpdateHealthConnectionSync(ctx, "user-1", latestSync); err != nil {
		t.Fatalf("UpdateHealthConnectionSync returned error: %v", err)
	}
	gotConnection, err = repo.GetHealthConnection(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetHealthConnection after sync returned error: %v", err)
	}
	if gotConnection.LastSyncedAt == nil || !gotConnection.LastSyncedAt.Equal(latestSync) {
		t.Fatalf("sync timestamp not saved: %+v", gotConnection)
	}
	if _, err := repo.GetHealthConnection(ctx, "missing-user"); err == nil {
		t.Fatal("expected missing connection error")
	}

	firstClaim := application.QuestClaim{
		ID:         "claim-1",
		UserID:     "user-1",
		Type:       "cardio",
		Title:      "Cardio Session",
		XP:         30,
		Stat:       "cardio",
		Source:     application.QuestClaimSourceGoogleHealth,
		SourceID:   "run-1",
		Evidence:   "Running · 35 min",
		OccurredAt: now,
		QuestDate:  "2026-07-17",
		Status:     application.QuestClaimStatusPending,
	}
	created, inserted, err := repo.UpsertQuestClaim(ctx, firstClaim)
	if err != nil {
		t.Fatalf("UpsertQuestClaim returned error: %v", err)
	}
	if !inserted || created.ID != "claim-1" || created.CreatedAt.IsZero() {
		t.Fatalf("unexpected inserted claim: inserted=%v claim=%+v", inserted, created)
	}

	duplicate := firstClaim
	duplicate.ID = "claim-duplicate"
	duplicate.SourceID = "run-duplicate"
	existing, inserted, err := repo.UpsertQuestClaim(ctx, duplicate)
	if err != nil {
		t.Fatalf("duplicate UpsertQuestClaim returned error: %v", err)
	}
	if inserted || existing.ID != "claim-1" {
		t.Fatalf("expected existing duplicate claim, inserted=%v claim=%+v", inserted, existing)
	}

	secondClaim := firstClaim
	secondClaim.ID = "claim-2"
	secondClaim.Type = "sleep"
	secondClaim.Title = "Sleep Goal Met"
	secondClaim.Stat = "recovery"
	secondClaim.SourceID = "sleep-1"
	secondClaim.Evidence = "450 minutes asleep"
	secondClaim.OccurredAt = now.Add(24 * time.Hour)
	secondClaim.QuestDate = "2026-07-18"
	if _, inserted, err := repo.UpsertQuestClaim(ctx, secondClaim); err != nil || !inserted {
		t.Fatalf("second UpsertQuestClaim inserted=%v err=%v", inserted, err)
	}

	pending, err := repo.ListPendingQuestClaims(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListPendingQuestClaims returned error: %v", err)
	}
	if len(pending) != 2 || pending[0].ID != "claim-1" || pending[1].ID != "claim-2" {
		t.Fatalf("unexpected pending claims: %+v", pending)
	}
	count, err := repo.CountPendingQuestClaims(ctx, "user-1")
	if err != nil || count != 2 {
		t.Fatalf("unexpected pending count=%d err=%v", count, err)
	}
	gotClaim, err := repo.GetQuestClaim(ctx, "user-1", "claim-1")
	if err != nil || gotClaim.Evidence != firstClaim.Evidence {
		t.Fatalf("unexpected fetched claim=%+v err=%v", gotClaim, err)
	}
	if _, err := repo.GetQuestClaim(ctx, "user-1", "missing-claim"); !errors.Is(err, application.ErrQuestClaimNotFound) {
		t.Fatalf("expected ErrQuestClaimNotFound, got %v", err)
	}

	claimedAt := now.Add(2 * time.Hour)
	if err := repo.MarkQuestClaimClaimed(ctx, "user-1", "claim-1", "activity-1", claimedAt); err != nil {
		t.Fatalf("MarkQuestClaimClaimed returned error: %v", err)
	}
	if err := repo.MarkQuestClaimClaimed(ctx, "user-1", "claim-1", "activity-1", claimedAt); !errors.Is(err, application.ErrQuestClaimAlreadyClaimed) {
		t.Fatalf("expected already claimed error, got %v", err)
	}
	gotClaim, err = repo.GetQuestClaim(ctx, "user-1", "claim-1")
	if err != nil {
		t.Fatalf("GetQuestClaim claimed returned error: %v", err)
	}
	if gotClaim.Status != application.QuestClaimStatusClaimed || gotClaim.ActivityID != "activity-1" || gotClaim.ClaimedAt == nil {
		t.Fatalf("claim was not marked claimed: %+v", gotClaim)
	}
	count, err = repo.CountPendingQuestClaims(ctx, "user-1")
	if err != nil || count != 1 {
		t.Fatalf("unexpected pending count after claim=%d err=%v", count, err)
	}
}

func TestRepositoryReturnsDatabaseErrorsAfterClose(t *testing.T) {
	ctx := context.Background()
	repo := NewRepository(testDB(t))
	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}
	sqlDB, err := repo.db.DB()
	if err != nil {
		t.Fatalf("unwrap db: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	now := time.Date(2026, 7, 17, 10, 0, 0, 0, time.UTC)
	claim := application.QuestClaim{
		ID:         "claim-1",
		UserID:     "user-1",
		Type:       "cardio",
		Title:      "Cardio Session",
		XP:         30,
		Stat:       "cardio",
		Source:     application.QuestClaimSourceGoogleHealth,
		OccurredAt: now,
		QuestDate:  "2026-07-17",
		Status:     application.QuestClaimStatusPending,
	}
	if err := repo.AutoMigrate(); err == nil {
		t.Fatal("expected AutoMigrate error after close")
	}
	if err := repo.SaveHealthAuthState(ctx, application.HealthAuthState{State: "state-1", UserID: "user-1", ExpiresAt: now.Add(time.Hour)}); err == nil {
		t.Fatal("expected SaveHealthAuthState error after close")
	}
	if _, err := repo.ConsumeHealthAuthState(ctx, "state-1", now); err == nil {
		t.Fatal("expected ConsumeHealthAuthState error after close")
	}
	if err := repo.SaveHealthConnection(ctx, application.HealthConnection{UserID: "user-1", RefreshToken: "refresh"}); err == nil {
		t.Fatal("expected SaveHealthConnection error after close")
	}
	if _, err := repo.GetHealthConnection(ctx, "user-1"); err == nil {
		t.Fatal("expected GetHealthConnection error after close")
	}
	if err := repo.UpdateHealthConnectionSync(ctx, "user-1", now); err == nil {
		t.Fatal("expected UpdateHealthConnectionSync error after close")
	}
	if _, _, err := repo.UpsertQuestClaim(ctx, claim); err == nil {
		t.Fatal("expected UpsertQuestClaim error after close")
	}
	if _, err := repo.ListPendingQuestClaims(ctx, "user-1"); err == nil {
		t.Fatal("expected ListPendingQuestClaims error after close")
	}
	if _, err := repo.CountPendingQuestClaims(ctx, "user-1"); err == nil {
		t.Fatal("expected CountPendingQuestClaims error after close")
	}
	if _, err := repo.GetQuestClaim(ctx, "user-1", "claim-1"); err == nil {
		t.Fatal("expected GetQuestClaim error after close")
	}
	if err := repo.MarkQuestClaimClaimed(ctx, "user-1", "claim-1", "activity-1", now); err == nil {
		t.Fatal("expected MarkQuestClaimClaimed error after close")
	}
}
