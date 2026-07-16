package main

import (
	"log"
	"net/http"

	"github.com/EDessin/MaxSelf/internal/platform/config"
	"github.com/EDessin/MaxSelf/internal/platform/database"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
	progressrest "github.com/EDessin/MaxSelf/internal/progress/adapters/inbound/rest"
	progresspostgres "github.com/EDessin/MaxSelf/internal/progress/adapters/outbound/postgres"
	"github.com/EDessin/MaxSelf/internal/progress/application"
)

func main() {
	cfg := config.Load("progress", "8083")
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	repo := progresspostgres.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		log.Fatal(err)
	}

	service := application.NewService(repo)
	handler := progressrest.NewHandler(service)

	log.Printf("progress service listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, httpx.WithCORS(handler.Routes(), cfg.FrontendURL)))
}
