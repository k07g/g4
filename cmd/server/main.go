package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/k07g/g4/internal/api"
	"github.com/k07g/g4/internal/auth"
	"github.com/k07g/g4/internal/config"
	"github.com/k07g/g4/internal/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()

	var provider auth.Provider
	switch cfg.AuthProvider {
	case config.AuthProviderMemory:
		log.Println("AUTH_PROVIDER=memory: using in-memory auth provider (local development only, not for production)")
		provider = auth.NewMemoryProvider()
	default:
		cognitoClient, err := auth.NewCognitoClient(ctx, cfg.AWSRegion, cfg.CognitoUserPoolID, cfg.CognitoClientID, cfg.CognitoClientSecret)
		if err != nil {
			log.Fatalf("failed to init cognito client: %v", err)
		}
		provider = cognitoClient
	}

	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	userRepo := db.NewUserRepository(database)
	handler := api.NewHandler(provider, userRepo)
	router := api.NewRouter(handler, api.AuthMiddleware(provider))

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("server listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
