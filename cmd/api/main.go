package main

import (
	"log"
	"net/http"

	facaderest "github.com/EDessin/MaxSelf/internal/facade/adapters/inbound/rest"
	facadepostgres "github.com/EDessin/MaxSelf/internal/facade/adapters/outbound/postgres"
	"github.com/EDessin/MaxSelf/internal/facade/application"
	"github.com/EDessin/MaxSelf/internal/platform/config"
	"github.com/EDessin/MaxSelf/internal/platform/database"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
)

func main() {
	cfg := config.Load("api", "8080")
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	repo := facadepostgres.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		log.Fatalf("migrate facade: %v", err)
	}
	healthClient := application.NewGoogleHealthHTTPClient(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleHealthRedirectURL, cfg.GoogleHealthAPIBaseURL, cfg.HTTPClientTimeout)
	service := application.NewServiceWithIntegrations(
		application.NewClient(cfg.IdentityServiceURL, cfg.HTTPClientTimeout),
		application.NewClient(cfg.ActivityServiceURL, cfg.HTTPClientTimeout),
		application.NewClient(cfg.ProgressServiceURL, cfg.HTTPClientTimeout),
		cfg.JWTSecret,
		repo,
		healthClient,
	)
	handler := facaderest.NewHandler(service, cfg)

	log.Printf("api facade listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, httpx.WithCORS(handler.Routes(), cfg.FrontendURL)))
}
