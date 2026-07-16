package main

import (
	"log"
	"net/http"

	activityrest "github.com/EDessin/MaxSelf/internal/activity/adapters/inbound/rest"
	activitypostgres "github.com/EDessin/MaxSelf/internal/activity/adapters/outbound/postgres"
	"github.com/EDessin/MaxSelf/internal/activity/application"
	"github.com/EDessin/MaxSelf/internal/platform/config"
	"github.com/EDessin/MaxSelf/internal/platform/database"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
)

func main() {
	cfg := config.Load("activity", "8082")
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	repo := activitypostgres.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		log.Fatal(err)
	}

	service := application.NewService(repo)
	handler := activityrest.NewHandler(service)

	log.Printf("activity service listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, httpx.WithCORS(handler.Routes(), cfg.FrontendURL)))
}
