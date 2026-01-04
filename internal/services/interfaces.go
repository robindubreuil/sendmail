package services

import (
	"context"
	"gosendmail/internal/models"
)

// EmailServiceInterface defines the interface for email services
type EmailServiceInterface interface {
	SendContactForm(ctx context.Context, form *models.ContactForm) error
	ValidateConfiguration() error
	HealthCheck() error
}

// RecaptchaServiceInterface defines the interface for reCAPTCHA services
type RecaptchaServiceInterface interface {
	Verify(ctx context.Context, response, remoteIP string) error
}

// NonceServiceInterface defines the interface for nonce services
type NonceServiceInterface interface {
	Generate() (string, error)
	Validate(nonce string) bool
	Shutdown()
}
