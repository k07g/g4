package config

import (
	"fmt"
	"os"
)

// AuthProvider selects the auth.Provider implementation used by the server.
type AuthProvider string

const (
	// AuthProviderCognito talks to a real Amazon Cognito user pool.
	AuthProviderCognito AuthProvider = "cognito"
	// AuthProviderMemory is an in-memory stand-in for local development and
	// testing that requires no AWS account.
	AuthProviderMemory AuthProvider = "memory"
)

type Config struct {
	Port                string
	AuthProvider        AuthProvider
	AWSRegion           string
	CognitoUserPoolID   string
	CognitoClientID     string
	CognitoClientSecret string
	DatabaseURL         string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		AuthProvider:        AuthProvider(getEnv("AUTH_PROVIDER", string(AuthProviderCognito))),
		AWSRegion:           os.Getenv("AWS_REGION"),
		CognitoUserPoolID:   os.Getenv("COGNITO_USER_POOL_ID"),
		CognitoClientID:     os.Getenv("COGNITO_CLIENT_ID"),
		CognitoClientSecret: os.Getenv("COGNITO_CLIENT_SECRET"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}

	switch cfg.AuthProvider {
	case AuthProviderCognito:
		if cfg.AWSRegion == "" {
			missing = append(missing, "AWS_REGION")
		}
		if cfg.CognitoUserPoolID == "" {
			missing = append(missing, "COGNITO_USER_POOL_ID")
		}
		if cfg.CognitoClientID == "" {
			missing = append(missing, "COGNITO_CLIENT_ID")
		}
	case AuthProviderMemory:
		// No external credentials required.
	default:
		return nil, fmt.Errorf("invalid AUTH_PROVIDER %q: must be %q or %q", cfg.AuthProvider, AuthProviderCognito, AuthProviderMemory)
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missing)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
