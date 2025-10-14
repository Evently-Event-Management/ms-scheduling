package services

import (
	"database/sql"
	"ms-scheduling/internal/config"
)

type SubscriberService struct {
	DB             *sql.DB
	KeycloakClient *KeycloakClient
	EmailService   *EmailService
	Config         *config.Config
}

func NewSubscriberService(db *sql.DB, keycloakClient *KeycloakClient, emailService *EmailService, cfg *config.Config) *SubscriberService {
	return &SubscriberService{
		DB:             db,
		KeycloakClient: keycloakClient,
		EmailService:   emailService,
		Config:         cfg,
	}
}
