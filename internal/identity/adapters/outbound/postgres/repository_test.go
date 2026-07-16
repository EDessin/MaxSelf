package postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/EDessin/MaxSelf/internal/identity/domain"
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

func TestRepositoryCreatesAndFindsUsers(t *testing.T) {
	repo := NewRepository(testDB(t))
	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}

	user := domain.User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo", AvatarURL: "avatar.png", PasswordHash: "hash"}
	identity := domain.AuthIdentity{ID: "identity-1", UserID: "user-1", Provider: domain.ProviderEmail, ProviderUserID: "demo@example.com", Email: "demo@example.com"}

	created, err := repo.CreateUser(context.Background(), user, identity)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	if created.ID != "user-1" || created.CreatedAt.IsZero() {
		t.Fatalf("unexpected created user: %+v", created)
	}

	byEmail, err := repo.FindByEmail(context.Background(), "demo@example.com")
	if err != nil {
		t.Fatalf("FindByEmail returned error: %v", err)
	}
	if byEmail.ID != "user-1" || byEmail.AvatarURL != "avatar.png" {
		t.Fatalf("unexpected email user: %+v", byEmail)
	}

	byProvider, err := repo.FindByProvider(context.Background(), domain.ProviderEmail, "demo@example.com")
	if err != nil {
		t.Fatalf("FindByProvider returned error: %v", err)
	}
	if byProvider.ID != "user-1" {
		t.Fatalf("unexpected provider user: %+v", byProvider)
	}

	if _, err := repo.FindByEmail(context.Background(), "missing@example.com"); err == nil {
		t.Fatal("expected missing email error")
	}
	if _, err := repo.FindByProvider(context.Background(), domain.ProviderGoogle, "missing"); err == nil {
		t.Fatal("expected missing provider error")
	}
}

func TestRepositoryLinksIdentity(t *testing.T) {
	repo := NewRepository(testDB(t))
	if err := repo.AutoMigrate(); err != nil {
		t.Fatalf("AutoMigrate returned error: %v", err)
	}
	user := domain.User{ID: "user-1", Email: "demo@example.com", DisplayName: "Demo"}
	emailIdentity := domain.AuthIdentity{ID: "identity-1", UserID: "user-1", Provider: domain.ProviderEmail, ProviderUserID: "demo@example.com", Email: "demo@example.com"}
	if _, err := repo.CreateUser(context.Background(), user, emailIdentity); err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	googleIdentity := domain.AuthIdentity{ID: "identity-2", UserID: "user-1", Provider: domain.ProviderGoogle, ProviderUserID: "google-1", Email: "demo@example.com"}
	if err := repo.LinkIdentity(context.Background(), googleIdentity); err != nil {
		t.Fatalf("LinkIdentity returned error: %v", err)
	}

	byProvider, err := repo.FindByProvider(context.Background(), domain.ProviderGoogle, "google-1")
	if err != nil {
		t.Fatalf("FindByProvider linked identity returned error: %v", err)
	}
	if byProvider.ID != "user-1" {
		t.Fatalf("unexpected linked provider user: %+v", byProvider)
	}
}
