package main

import (
	"log"
	"net/http"

	facaderest "github.com/EDessin/MaxSelf/internal/facade/adapters/inbound/rest"
	"github.com/EDessin/MaxSelf/internal/facade/application"
	"github.com/EDessin/MaxSelf/internal/platform/config"
	"github.com/EDessin/MaxSelf/internal/platform/httpx"
)

func main() {
	cfg := config.Load("api", "8080")
	service := application.NewService(
		application.NewClient(cfg.IdentityServiceURL, cfg.HTTPClientTimeout),
		application.NewClient(cfg.ActivityServiceURL, cfg.HTTPClientTimeout),
		application.NewClient(cfg.ProgressServiceURL, cfg.HTTPClientTimeout),
		cfg.JWTSecret,
	)
	handler := facaderest.NewHandler(service, cfg)

	log.Printf("api facade listening on :%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, httpx.WithCORS(handler.Routes(), cfg.FrontendURL)))
}
