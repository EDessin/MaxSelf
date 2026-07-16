package main

import (
	"log"
	"net/http"

	identityrest "github.com/EDessin/MaxSelf/internal/identity/adapters/inbound/rest"
	identitypostgres "github.com/EDessin/MaxSelf/internal/identity/adapters/outbound/postgres"
	"github.com/EDessin/MaxSelf/internal/identity/application"
	"github.com/EDessin/MaxSelf/internal/platform/config"
	"github.com/EDessin/MaxSelf/internal/platform/database"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
)

func main() {
	cfg := config.Load("identity", "8081")
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	repo := identitypostgres.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		log.Fatal(err)
	}

	service := application.NewService(repo, cfg.JWTSecret, cfg.AccessTokenExpiresIn)
	handler := identityrest.NewHandler(service, cfg)

	log.Printf("identity service listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, httpx.WithCORS(handler.Routes(), cfg.FrontendURL)))
}
