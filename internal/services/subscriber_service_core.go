package services

import "database/sql"

type SubscriberService struct {
	DB             *sql.DB
	KeycloakClient *KeycloakClient
	EmailService   *EmailService
}

func NewSubscriberService(db *sql.DB, keycloakClient *KeycloakClient, emailService *EmailService) *SubscriberService {
	return &SubscriberService{
		DB:             db,
		KeycloakClient: keycloakClient,
		EmailService:   emailService,
	}
}
